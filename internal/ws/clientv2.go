// Package ws
package ws

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"time"

	"rplatform-echo/cmd/web"

	"github.com/coder/websocket"
	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
)

var (
	writeTimeout         = 30 * time.Second // per-write timeout to client
	subscriberBufferSize = 32               // buffered messages per client
)

type Client struct {
	conn *websocket.Conn
	send chan []byte
	hub  *Room

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
		var msgMap map[string]any
		if err := json.Unmarshal(msg, &msgMap); err != nil {
			// send normal message if decode errors
			c.hub.broadcast <- msg
		} else {
			msgMap["sender_id"] = c.userID
			msgMap["email"] = c.email
			_msgContent, ok := msgMap["chat_message"].(string)
			var msgContent string
			if ok {
				msgContent = _msgContent
			}
			msgToSend, err := json.Marshal(msgMap)
			if err != nil {
				c.hub.broadcast <- msg
			} else {
				c.hub.broadcast <- msgToSend
			}
			// insert data to db here
			if _, err := c.hub.manager.messageSvc.Create(ctx, c.hub.id, c.userID, msgContent); err != nil {
				log.Println(err.Error())
			}
		}
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

	var buf bytes.Buffer

	for {
		select {
		case msg, ok := <-c.send:
			if !ok {
				return
			}
			buf.Reset()

			var msgToSend map[string]any
			if err := json.Unmarshal(msg, &msgToSend); err != nil {
				log.Println("Error decoding", err)
			}
			err := web.ChatMessage(msgToSend["email"].(string), msgToSend["chat_message"].(string), c.userID, msgToSend["sender_id"].(string)).Render(context.Background(), &buf)
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

func ServeWs(hub *Room, c echo.Context) error {
	w := c.Response().Writer
	r := c.Request()
	conn, err := websocket.Accept(w, r, nil)
	if err != nil {
		return err
	}
	user := c.Get("user").(*jwt.Token)
	claims := user.Claims.(jwt.MapClaims)
	userID := claims["user_id"].(string)
	email := claims["email"].(string)

	client := &Client{
		hub:    hub,
		conn:   conn,
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
