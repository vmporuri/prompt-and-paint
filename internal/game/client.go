package game

import (
	"context"
	"encoding/json"
	"log"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/lithammer/shortuuid"
	"github.com/redis/go-redis/v9"
)

type Client struct {
	Conn      *websocket.Conn
	UserID    string
	Username  string
	RoomID    string
	Pubsub    *redis.PubSub
	Mutex     *sync.RWMutex
	WriteChan chan []byte
	Ctx       context.Context
	Cancel    context.CancelFunc
}

type GameMessage struct {
	Headers map[string]any `json:"HEADERS"`
	Event   gameEvent      `json:"event"`
	Msg     string         `json:"msg"`
}

func NewClient(conn *websocket.Conn) *Client {
	ctx, cancel := context.WithCancel(context.Background())
	return &Client{
		Conn:      conn,
		WriteChan: make(chan []byte),
		Mutex:     &sync.RWMutex{},
		Ctx:       ctx,
		Cancel:    cancel,
	}
}

func DispatchGameEvent(client *Client, gameMsg *GameMessage) {
	switch gameMsg.Event {
	case create:
		client.handleCreate()
	case join:
		client.handleJoin(gameMsg)
	case username:
		client.handleUsername(gameMsg)
	case ready:
		client.handleReady()
	case prompt:
		client.handlePrompt(gameMsg)
	case pickPicture:
		client.handlePicture(gameMsg)
	case vote:
		client.handleVote(gameMsg)
	case CloseWS:
		client.handleClose()
	}
}

func (c *Client) readPump() {
	defer c.Pubsub.Close()

	ch := c.Pubsub.Channel()

	for {
		select {
		case msg := <-ch:
			psEvent := PSMessage{}
			err := json.Unmarshal([]byte(msg.Payload), &psEvent)
			if err != nil {
				log.Printf("Error unmarshalling pubsub message: %v", err)
			}

			switch psEvent.Event {
			case newPlayerList:
				c.updatePlayerList([]byte(psEvent.Msg))
			case enterGame:
				c.loadGame([]byte(psEvent.Msg))
			case votePage:
				c.displayCandidates([]byte(psEvent.Msg))
			case leaderboard:
				c.displayLeaderboard([]byte(psEvent.Msg))
			}
		case <-c.Ctx.Done():
			return
		}
	}
}

func (c *Client) handleCreate() {
	room, err := createRoom()
	if err != nil {
		log.Printf("Error creating new room: %f", err)
		return
	}
	c.Mutex.Lock()
	c.RoomID = room.ID
	c.Mutex.Unlock()
	subscribeClient(c)
	usernamePage, err := generateUsername()
	if err != nil {
		log.Printf("Error creating username page template: %v", err)
		return
	}
	c.WriteChan <- usernamePage
}

func (c *Client) handleJoin(gameMsg *GameMessage) {
	roomID := gameMsg.Msg
	exists := checkMembershipRedisSet(c.Ctx, roomList, roomID)
	if !exists {
		log.Printf("Room %s does not exist", gameMsg.Msg)
		return
	}
	c.Mutex.Lock()
	c.RoomID = roomID
	c.Mutex.Unlock()
	subscribeClient(c)
	usernamePage, err := generateUsername()
	if err != nil {
		log.Printf("Error creating username page template: %v", err)
		return
	}
	c.WriteChan <- usernamePage
}

func (c *Client) handleUsername(gameMsg *GameMessage) {
	c.Mutex.Lock()
	c.UserID = shortuuid.New()
	c.Username = gameMsg.Msg
	c.Mutex.Unlock()
	newUserMsg, err := json.Marshal(
		newPSMessageWithOptMsg(newUser, c.UserID, c.Username),
	)
	if err != nil {
		log.Printf("Error encoding new user message: %v", err)
		return
	}
	err = publishClientMessage(c, newUserMsg)
	if err != nil {
		log.Printf("Error publishing new username: %v", err)
		return
	}
	wpd := &waitingPageData{RoomID: c.RoomID}
	waitingPage, err := generateWaitingPage(wpd)
	if err != nil {
		log.Printf("Error creating waiting page template: %v", err)
		return
	}
	c.WriteChan <- waitingPage
}

func (c *Client) handleReady() {
	readyMsg, err := json.Marshal(newPSMessage(ready, c.UserID))
	if err != nil {
		log.Printf("Error encoding new user message: %v", err)
		return
	}
	err = publishClientMessage(c, readyMsg)
	if err != nil {
		log.Printf("Error updating player status to ready: %v", err)
		return
	}
}

func (c *Client) updatePlayerList(players []byte) {
	c.WriteChan <- players
}

func (c *Client) loadGame(gamePage []byte) {
	c.WriteChan <- gamePage
}

func (c *Client) handleClose() {
	closeMsg, err := json.Marshal(newPSMessage(CloseWS, c.UserID))
	if err != nil {
		log.Printf("Error encoding close message: %v", err)
		return
	}
	log.Printf("User %s disconnected", c.UserID)
	err = publishClientMessage(c, closeMsg)
	if err != nil {
		log.Printf("Error publishing close message: %v", err)
		return
	}
	c.Cancel()
}

func (c *Client) handlePrompt(gameMsg *GameMessage) {
	url, err := generatePicture(c, gameMsg.Msg)
	if err != nil {
		log.Printf("Error generating image: %v", err)
		return
	}
	ipd := &imagePreviewData{URL: url}
	picturePreview, err := generatePicturePreview(ipd)
	if err != nil {
		log.Printf("Error creating picture preview template: %v", err)
		return
	}
	c.WriteChan <- picturePreview
}

func (c *Client) displayCandidates(candidates []byte) {
	c.WriteChan <- candidates
}

func (c *Client) handlePicture(gameMsg *GameMessage) {
	err := setRedisHash(c.Ctx, c.UserID, string(picture), gameMsg.Msg)
	if err != nil {
		log.Printf("Error storing user prompt: %v", err)
		return
	}
	sentPrompt, err := json.Marshal(newPSMessage(picture, gameMsg.Msg))
	if err != nil {
		log.Printf("Error encoding user prompt: %v", err)
		return
	}
	err = publishClientMessage(c, sentPrompt)
	if err != nil {
		log.Printf("Error publishing generated image url: %v", err)
		return
	}
}

func (c *Client) handleVote(gameMsg *GameMessage) {
	vote, err := json.Marshal(newPSMessage(gameMsg.Event, gameMsg.Msg))
	if err != nil {
		log.Printf("Error encoding vote: %v", err)
		return
	}
	err = publishClientMessage(c, vote)
	if err != nil {
		log.Printf("Error publishing player vote: %v", err)
		return
	}
}

func (c *Client) displayLeaderboard(leaderboard []byte) {
	c.WriteChan <- leaderboard
}
