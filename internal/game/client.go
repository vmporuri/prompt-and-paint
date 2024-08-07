// Package game implements the Prompt to Paint game by handling game events,
// relaying information between rooms and clients, and synchronizing clients.
package game

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
)

// Represents a single user of the website. Associated with one WebSocket connection.
// Acts as a middle-man for all incoming and outgoing messages.
// Uniquely identified by UserID. Username does not have to be unique.
// Connected to room specified by RoomID and communicates with Pubsub.
type Client struct {
	Conn      *websocket.Conn
	UserID    string
	Username  string
	RoomID    string
	Pubsub    *redis.PubSub
	Mutex     *sync.Mutex
	WriteChan chan []byte
	Ctx       context.Context
	Cancel    context.CancelFunc
}

// A data structure that holds user input in the game.
// Matches the structure of HTMX WebSocket messages and form data specified in the templates.
type GameMessage struct {
	Headers map[string]any `json:"HEADERS"`
	Event   gameEvent      `json:"event"`
	Msg     string         `json:"msg"`
}

// Creates a new client with the provided userID.
// Automatically reconnects to game if existing userID is found.
func NewClient(conn *websocket.Conn, userID string) *Client {
	ctx, cancel := context.WithCancel(context.Background())
	client := &Client{
		Conn:      conn,
		UserID:    userID,
		WriteChan: make(chan []byte),
		Mutex:     &sync.Mutex{},
		Ctx:       ctx,
		Cancel:    cancel,
	}
	username, err := getRedisHash(client.Ctx, userID, string(username))
	if err != nil {
		log.Printf("Created new user: %s", userID)
		return client
	}
	roomID, err := getRedisHash(client.Ctx, userID, string(roomID))
	if err != nil {
		log.Printf("Created new user: %s", userID)
		return client
	}

	client.Username = username
	err = client.joinRoom(roomID)
	if err != nil {
		log.Printf("Error unable to reconnect to room: %v", err)
	}
	err = client.reconnectClient()
	if err != nil {
		log.Printf("Error unable to reconnect to room: %v", err)
	}
	return client
}

// Reconnects client to the room by updating room's information and fetching room state.
func (c *Client) reconnectClient() error {
	reconnectionMessage := &PSMessage{
		Event:  reconnect,
		Sender: c.UserID,
		Msg:    c.Username,
	}
	reconnectionJSON, err := json.Marshal(reconnectionMessage)
	if err != nil {
		return err
	}
	err = publishClientMessage(c, reconnectionJSON)
	if err != nil {
		return err
	}
	roomState, err := c.fetchRoomState()
	if err != nil {
		return errors.New("Unable to fetch current room state")
	}
	go func() {
		c.WriteChan <- []byte(roomState)
	}()
	return nil
}

// Adds client to the room identified by roomID.
// Returns a non-nil error if the room does not exist.
func (c *Client) joinRoom(roomID string) error {
	exists, err := roomRepo.lookupRoom(c.Ctx, roomID)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("Room does not exist")
	}
	c.Mutex.Lock()
	c.RoomID = roomID
	c.Mutex.Unlock()
	subscribeClient(c)
	return nil
}

// Fetches room state in case of reconnection or any other error.
func (c *Client) fetchRoomState() (string, error) {
	return getRedisHash(c.Ctx, c.RoomID, string(roomBackup))
}

// Dispatches the appropriate event handler for the given gameMsg.
func DispatchGameEvent(client *Client, gameMsg *GameMessage) {
	switch gameMsg.Event {
	case create:
		go client.handleCreate()
	case join:
		go client.handleJoin(gameMsg)
	case setUsername:
		go client.handleUsername(gameMsg)
	case ready:
		go client.handleReady()
	case prompt:
		go client.handlePrompt(gameMsg)
	case pickPicture:
		go client.handlePicture(gameMsg)
	case vote:
		go client.handleVote(gameMsg)
	case leave:
		go client.handleLeave()
	case CloseWS:
		go client.handleClose()
	}
}

// Continually reads in all incoming messages from the pub/sub communication line.
// Dispatches the appropriate event handler for the incoming internal event.
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
				go c.updatePlayerList([]byte(psEvent.Msg))
			case enterGame:
				go c.loadGame([]byte(psEvent.Msg))
			case votePage:
				go c.displayCandidates([]byte(psEvent.Msg))
			case sendLeaderboard:
				go c.displayLeaderboard([]byte(psEvent.Msg))
			}
		case <-c.Ctx.Done():
			return
		}
	}
}

// Marks a player as "ready" in the database (so the room can see).
func (c *Client) readyPlayer() error {
	return setRedisHash(c.Ctx, c.UserID, string(ready), string(isReady))
}

// Marks a player as "not ready" in the database.
func (c *Client) unreadyPlayer() error {
	return setRedisHash(c.Ctx, c.UserID, string(ready), string(isNotReady))
}

// Uploads user data to the database.
// Used to persist player data in case of unexpected WebSocket disconnections.
func (c *Client) backupClientData() error {
	err := setRedisHash(c.Ctx, c.UserID, string(roomID), c.RoomID)
	if err != nil {
		return errors.New("Error backing up room id")
	}
	err = setRedisHash(c.Ctx, c.UserID, string(username), c.Username)
	if err != nil {
		return errors.New("Error backing up username")
	}
	return nil
}

// Deletes a client backup. Used when a user sends an exit signal.
func (c *Client) deleteClientBackup() error {
	return deleteRedisKey(c.Ctx, c.UserID)
}

// Creates a new room and sends the user to the username page.
func (c *Client) handleCreate() {
	room, err := createRoom()
	if err != nil {
		log.Printf("Error creating new room: %f", err)
		return
	}
	err = c.joinRoom(room.ID)
	if err != nil {
		log.Printf("Error joining room: %v", err)
	}
	usernamePage, err := generateUsername()
	if err != nil {
		log.Printf("Error creating username page template: %v", err)
		return
	}
	c.WriteChan <- usernamePage
}

// Connects the user to the specified room (if it exists) and sends the user to
// the username page.
func (c *Client) handleJoin(gameMsg *GameMessage) {
	roomID := gameMsg.Msg
	err := c.joinRoom(roomID)
	if err != nil {
		log.Printf("Error joining room: %v", err)
	}
	usernamePage, err := generateUsername()
	if err != nil {
		log.Printf("Error creating username page template: %v", err)
		return
	}
	c.WriteChan <- usernamePage
}

// Accepts the player's chosen username and sends the user to the waiting room.
func (c *Client) handleUsername(gameMsg *GameMessage) {
	c.Mutex.Lock()
	c.Username = gameMsg.Msg
	c.Mutex.Unlock()
	err := setRedisHash(c.Ctx, c.UserID, string(ready), false)
	if err != nil {
		log.Printf("Error initializing player status: %v", err)
	}
	newUserMsg, err := json.Marshal(
		newPSMessage(newUser, c.UserID, c.Username),
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

// Marks the player as ready to start the next round.
func (c *Client) handleReady() {
	err := c.readyPlayer()
	if err != nil {
		log.Printf("Error setting player status to ready: %v", err)
		return
	}
	readyMsg, err := json.Marshal(newPSMessage(ready, c.UserID, c.UserID))
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

// Sends the updated player list after a new user joins.
func (c *Client) updatePlayerList(players []byte) {
	c.WriteChan <- players
}

// Sends the user to the game page. Called after all players have been marked as ready.
func (c *Client) loadGame(gamePage []byte) {
	err := c.backupClientData()
	if err != nil {
		log.Println(err)
	}
	err = c.unreadyPlayer()
	if err != nil {
		log.Printf("Error setting player status to unready: %v", err)
		return
	}
	c.WriteChan <- gamePage
}

// Handles a user submitted exit signal.
// Unlike handleClose, handleLeave deletes all user backups.
func (c *Client) handleLeave() {
	err := c.deleteClientBackup()
	if err != nil {
		log.Printf("Error deleting client backup data: %v", err)
	}
	c.Conn.Close()
}

// Handles an unexpected WebSocket disconnection by halting all goroutines associated
// with the client.
func (c *Client) handleClose() {
	defer c.Cancel()
	defer close(c.WriteChan)

	closeMsg, err := json.Marshal(newPSMessage(CloseWS, c.UserID, c.UserID))
	if err != nil {
		log.Printf("Error encoding close message: %v", err)
	}
	log.Printf("User %s disconnected", c.UserID)
	err = publishClientMessage(c, closeMsg)
	if err != nil {
		log.Printf("Error publishing close message: %v", err)
	}
}

// Handles the user submitted prompt by generating a picture and sending it back.
// Note that prompts that OpenAI content violations will not generate a picture.
func (c *Client) handlePrompt(gameMsg *GameMessage) {
	url, err := generateAIPicture(c.Ctx, gameMsg.Msg)
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

// Accepts the user's chosen picture. Stores in database and relays to the room.
func (c *Client) handlePicture(gameMsg *GameMessage) {
	err := c.readyPlayer()
	if err != nil {
		log.Printf("Error setting player status to ready: %v", err)
		return
	}
	err = setRedisHash(c.Ctx, c.UserID, string(picture), gameMsg.Msg)
	if err != nil {
		log.Printf("Error storing user prompt: %v", err)
		return
	}
	sentPrompt, err := json.Marshal(newPSMessage(getPicture, c.UserID, gameMsg.Msg))
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

// Displays all of the client submissions for the room.
func (c *Client) displayCandidates(candidates []byte) {
	err := c.unreadyPlayer()
	if err != nil {
		log.Printf("Error setting player status to unready: %v", err)
		return
	}
	c.WriteChan <- candidates
}

// Handles the players vote by updating the database and relaying to the room.
func (c *Client) handleVote(gameMsg *GameMessage) {
	err := c.readyPlayer()
	if err != nil {
		log.Printf("Error setting player status to ready: %v", err)
		return
	}
	vote, err := json.Marshal(newPSMessage(gameMsg.Event, c.UserID, gameMsg.Msg))
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

// Displays the current leaderboard, which includes round scores and total scores.
func (c *Client) displayLeaderboard(leaderboard []byte) {
	err := c.unreadyPlayer()
	if err != nil {
		log.Printf("Error setting player status to unready: %v", err)
		return
	}
	c.WriteChan <- leaderboard
}
