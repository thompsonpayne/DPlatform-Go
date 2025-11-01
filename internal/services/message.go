package services

import (
	"context"
	"errors"

	"rplatform-echo/internal/repository"

	"github.com/google/uuid"
)

type MessageService struct {
	q *repository.Queries
}

func NewMessageService(q *repository.Queries) *MessageService {
	return &MessageService{
		q: q,
	}
}

func (m *MessageService) List(ctx context.Context, roomID string) ([]repository.GetAllMessagesRow, error) {
	return m.q.GetAllMessages(ctx, roomID)
}

func (m *MessageService) Create(ctx context.Context, roomID string, userID string, content string) (repository.Message, error) {
	err := checkValidRequest(roomID, userID)
	if err != nil {
		return repository.Message{}, err
	}

	return m.q.CreateMessage(ctx, repository.CreateMessageParams{
		RoomID:  roomID,
		UserID:  userID,
		ID:      uuid.NewString(),
		Content: content,
	})
}

func (m *MessageService) Delete(ctx context.Context, roomID string, userID string, messageID string) error {
	err := checkValidRequest(roomID, userID)
	if err != nil {
		return err
	}
	return m.q.DeleteMessage(ctx, messageID)
}

func checkValidRequest(roomID string, userID string) error {
	if roomID == "" {
		return errors.New("roomID can't be empty")
	}

	if userID == "" {
		return errors.New("userID can't be empty")
	}
	return nil
}
