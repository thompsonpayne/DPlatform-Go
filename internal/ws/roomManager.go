package ws

import (
	"context"
	"errors"
	"log"
	"sync"

	"rplatform-echo/internal/services"
)

type RoomManager struct {
	rooms map[string]*Room
	mu    sync.RWMutex

	messageSvc *services.MessageService
}

func NewRoomManager(messageSvc *services.MessageService) *RoomManager {
	return &RoomManager{
		rooms: make(map[string]*Room),

		messageSvc: messageSvc,
	}
}

func (m *RoomManager) GetRoom(roomID string) *Room {
	m.mu.RLock()
	room, ok := m.rooms[roomID]
	m.mu.RUnlock()
	if ok {
		return room
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if room, ok := m.rooms[roomID]; ok {
		return room
	}

	room = NewRoom(roomID, m)
	m.rooms[roomID] = room
	log.Println("New chat room created: ", roomID)

	go room.Run(context.Background())
	return room
}

func (m *RoomManager) RemoveRoom(roomID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.mu.RLock()
	if _, ok := m.rooms[roomID]; ok {
		delete(m.rooms, roomID)
	} else {
		return errors.New("Room does not exist")
	}
	return nil
}
