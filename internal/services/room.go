// Package services
package services

import (
	"context"
	"errors"

	"rplatform-echo/internal/repository"

	"github.com/oklog/ulid/v2"
)

// RoomService encapsulates room-related business logic.
type RoomService struct {
	q *repository.Queries
}

func NewRoomService(q *repository.Queries) *RoomService { return &RoomService{q: q} }

// List returns all rooms ordered by created_at desc.
func (s *RoomService) List(ctx context.Context) ([]repository.Room, error) {
	return s.q.GetAllRooms(ctx)
}

// Create creates a room with the given name.
func (s *RoomService) Create(ctx context.Context, name string) (repository.Room, error) {
	if name == "" {
		return repository.Room{}, errors.New("name is required")
	}
	params := repository.CreateRoomParams{ID: ulid.Make().String(), Name: name}
	return s.q.CreateRoom(ctx, params)
}

// Delete deletes a room by id.
func (s *RoomService) Delete(ctx context.Context, id string) error {
	if id == "" {
		return errors.New("id is required")
	}
	return s.q.DeleteRoom(ctx, id)
}

// Get returns a single room by id.
func (s *RoomService) Get(ctx context.Context, id string) (repository.Room, error) {
	if id == "" {
		return repository.Room{}, errors.New("id is required")
	}
	return s.q.GetRoom(ctx, id)
}

// Update a room
func (s *RoomService) Update(ctx context.Context, id string, name string) error {
	if id == "" {
		return errors.New("id is required")
	}
	return s.q.UpdateRoom(ctx, repository.UpdateRoomParams{ID: id, Name: name})
}
