package game

import (
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/lithammer/shortuuid"
	"github.com/redis/go-redis/v9"
)

type Client struct {
	Conn     *websocket.Conn
	UserID   string
	Username string
	RoomID   string
	Pubsub   *redis.PubSub
	Mutex    *sync.RWMutex
}

type GameMessage struct {
	Headers map[string]any `json:"HEADERS"`
	Event   gameEvent      `json:"event"`
	Msg     string         `json:"msg"`
}

func NewClient(conn *websocket.Conn) *Client {
	return &Client{Conn: conn, Mutex: &sync.RWMutex{}}
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
	}
}

func (c *Client) readPump() {
	defer c.Pubsub.Close()

	ch := c.Pubsub.Channel()

	for msg := range ch {
		payload := strings.Split(msg.Payload, ":")
		event := payload[0]
		msg := payload[1]

		switch gameEvent(event) {
		case newPlayerList:
			c.updatePlayerList([]byte(msg))
		case enterGame:
			c.loadGame([]byte(msg))
		}
	}
}

func (c *Client) handleCreate() {
	room := createRoom()
	c.Mutex.Lock()
	c.RoomID = room.ID
	c.Mutex.Unlock()
	subscribeClient(c)

	err := c.Conn.WriteMessage(websocket.TextMessage, generateUsername())
	if err != nil {
		log.Println("Could not send username template")
	}
}

func (c *Client) handleJoin(gameMsg *GameMessage) {
	roomID := gameMsg.Msg
	exists := checkMembershipRedisSet(roomList, roomID)
	if !exists {
		log.Printf("Room %s does not exist", gameMsg.Msg)
		return
	}
	c.Mutex.Lock()
	c.RoomID = roomID
	c.Mutex.Unlock()
	subscribeClient(c)

	err := c.Conn.WriteMessage(websocket.TextMessage, generateUsername())
	if err != nil {
		log.Println("Could not send username template")
	}
}

func (c *Client) handleUsername(gameMsg *GameMessage) {
	c.Mutex.Lock()
	c.UserID = shortuuid.New()
	c.Username = gameMsg.Msg
	c.Mutex.Unlock()
	publishClientMessage(c, fmt.Sprintf("new-user:%s-%s", c.UserID, c.Username))
	err := c.Conn.WriteMessage(websocket.TextMessage, generateWaitingPage(c))
	if err != nil {
		log.Println("Could not send username template")
	}
}

func (c *Client) handleReady() {
	setRedisKey(c.UserID, ready)
	publishClientMessage(c, fmt.Sprintf("ready:%s", c.UserID))
}

func (c *Client) updatePlayerList(players []byte) {
	err := c.Conn.WriteMessage(websocket.TextMessage, players)
	if err != nil {
		log.Println("Could not send player-list template")
	}
}

func (c *Client) loadGame(gamePage []byte) {
	err := c.Conn.WriteMessage(websocket.TextMessage, gamePage)
	if err != nil {
		log.Println("Could not send game page template")
	}
}
