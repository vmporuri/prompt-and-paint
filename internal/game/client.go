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
				log.Println("Could not unmarshal pubsub message")
			}

			switch psEvent.Event {
			case newPlayerList:
				c.updatePlayerList([]byte(psEvent.Msg))
			case enterGame:
				c.loadGame([]byte(psEvent.Msg))
			}
		case <-c.Ctx.Done():
			return
		}
	}
}

func (c *Client) handleCreate() {
	room := createRoom()
	c.Mutex.Lock()
	c.RoomID = room.ID
	c.Mutex.Unlock()
	subscribeClient(c)
	c.WriteChan <- generateUsername()
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
	c.WriteChan <- generateUsername()
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
		log.Println("Could not encode new user message")
		return
	}
	publishClientMessage(c, newUserMsg)
	c.WriteChan <- generateWaitingPage(c)
}

func (c *Client) handleReady() {
	setRedisKey(c.Ctx, c.UserID, ready)
	readyMsg, err := json.Marshal(newPSMessage(ready, c.UserID))
	if err != nil {
		log.Println("Could not encode new user message")
		return
	}
	publishClientMessage(c, readyMsg)
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
		log.Println("Could not encode new user message")
		return
	}
	publishClientMessage(c, closeMsg)
	c.Cancel()
}
