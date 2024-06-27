package game

import (
	"fmt"
	"log"
	"strings"

	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
)

type Client struct {
	Conn     *websocket.Conn
	Username string
	RoomID   string
	Pubsub   *redis.PubSub
}

type GameMessage struct {
	Headers map[string]any `json:"HEADERS"`
	Event   gameEvent      `json:"event"`
	Msg     string         `json:"msg"`
}

type gameEvent string

const (
	create   gameEvent = "create-room"
	join     gameEvent = "join-room"
	username gameEvent = "set-username"
	ready    gameEvent = "ready"
)

const (
	newPlayerList string = "new-player-list"
)

func DispatchGameEvent(client *Client, gameMsg *GameMessage) {
	switch gameMsg.Event {
	case create:
		client.handleCreate()
	case join:
		client.handleJoin(gameMsg)
	case username:
		client.handleUsername(gameMsg)
	case ready:
		// handleReady(client, gameMsg)
	}
}

func (c *Client) readPump() {
	defer c.Pubsub.Close()

	ch := c.Pubsub.Channel()

	for msg := range ch {
		c.updatePlayerList([]byte(msg.Payload))
		log.Println(msg.Channel, msg.Payload)
		payload := strings.Split(msg.Payload, ":")
		event := payload[0]
		msg := payload[1]

		switch event {
		case newPlayerList:
			c.updatePlayerList([]byte(msg))
		}
	}
}

func (c *Client) handleCreate() {
	room := createRoom()
	c.RoomID = room.ID
	subscribeClient(c)

	err := c.Conn.WriteMessage(websocket.TextMessage, generateUsername())
	if err != nil {
		log.Println("Could not send username template")
	}
}

func (client *Client) handleJoin(gameMsg *GameMessage) {
	room, exists := roomList[gameMsg.Msg]
	if !exists {
		log.Printf("Room %s does not exist", gameMsg.Msg)
		return
	}
	client.RoomID = room.ID
	subscribeClient(client)

	err := client.Conn.WriteMessage(websocket.TextMessage, generateUsername())
	if err != nil {
		log.Println("Could not send username template")
	}
}

func (c *Client) handleUsername(gameMsg *GameMessage) {
	c.Username = gameMsg.Msg
	publishClientMessage(c, fmt.Sprintf("new-user:%s", c.Username))
	err := c.Conn.WriteMessage(websocket.TextMessage, generateWaitingRoom(c))
	if err != nil {
		log.Println("Could not send username template")
	}
}

func (c *Client) updatePlayerList(players []byte) {
	err := c.Conn.WriteMessage(websocket.TextMessage, players)
	if err != nil {
		log.Println("Could not send player-list template")
	}
}
