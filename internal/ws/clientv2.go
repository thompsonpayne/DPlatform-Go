// Package ws
package ws

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"rplatform-echo/cmd/web"

	"github.com/coder/websocket"
	"github.com/golang-jwt/jwt/v5"
)

var (
	writeTimeout         = 30 * time.Second // per-write timeout to client
	subscriberBufferSize = 32               // buffered messages per client
)

type Client struct {
	conn *websocket.Conn
	send chan []byte
	hub  *Hub

	userID string
	email  string
}

func (c *Client) readPump() {
	ctx := context.Background()
	defer func() {
		c.hub.unregister <- c
		if err := c.conn.CloseNow(); err != nil {
			log.Println("Error closing connection")
		}
	}()

	for {
		_, msg, err := c.conn.Read(ctx)
		if err != nil {
			if websocket.CloseStatus(err) == websocket.StatusNormalClosure || websocket.CloseStatus(err) == websocket.StatusGoingAway {
				// normal close
				return
			}
			log.Printf("Error reading ws %v", err)
			return
		}
		c.hub.broadcast <- msg
	}
}

func (c *Client) writePump() {
	// ctx := context.Background()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	defer func() {
		if err := c.conn.CloseNow(); err != nil {
			log.Println("Error closing connection")
		}
		c.hub.unregister <- c
	}()

	for {
		select {
		case msg, ok := <-c.send:
			if !ok {
				return
			}
			var msgToSend map[string]any
			if err := json.Unmarshal(msg, &msgToSend); err != nil {
				log.Println("Error decoding", err)
			}
			msgToSend["userID"] = c.userID
			msgToSend["email"] = c.email
			// encodedMsg, err := json.Marshal(msgToSend)
			var buf bytes.Buffer
			err := web.ChatMessage(c.email, msgToSend["chat_message"].(string)).Render(context.Background(), &buf)
			if err != nil {
				log.Println("Error encoding", err)
			}
			html := buf.Bytes()
			if err := writeWithTimeout(ctx, writeTimeout, c.conn, html, websocket.MessageText); err != nil {
				log.Printf("Error writing ws %v", err)
			}
		case <-ctx.Done():
			return
		}
	}
}

func writeWithTimeout(ctx context.Context, timeout time.Duration, conn *websocket.Conn, msg []byte, typ websocket.MessageType) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return conn.Write(ctx, typ, msg)
}

func ServeWs(hub *Hub, w http.ResponseWriter, r *http.Request, claims jwt.MapClaims) error {
	c, err := websocket.Accept(w, r, nil)
	if err != nil {
		return err
	}
	userID := claims["user_id"].(string)
	email := claims["email"].(string)

	client := &Client{
		hub:    hub,
		conn:   c,
		send:   make(chan []byte, subscriberBufferSize),
		userID: userID,
		email:  email,
	}
	log.Println("Client is registering", email)

	client.hub.register <- client

	go client.writePump()
	go client.readPump()
	return nil
}
