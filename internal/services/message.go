package services

import (
	"context"
	"database/sql"
	"errors"

	"rplatform-echo/internal/repository"

	"github.com/oklog/ulid/v2"
)

type MessageService struct {
	q *repository.Queries
}

func NewMessageService(q *repository.Queries) *MessageService {
	return &MessageService{
		q: q,
	}
}

func (m *MessageService) ListFirst(ctx context.Context, roomID string) ([]repository.GetInitalMessagesRow, error) {
	return m.q.GetInitalMessages(ctx, roomID)
}

func (m *MessageService) ListNext(ctx context.Context, roomID string, createdAt sql.NullTime) ([]repository.GetPaginatedMessagesRow, error) {
	return m.q.GetPaginatedMessages(ctx, repository.GetPaginatedMessagesParams{
		RoomID:   roomID,
		Datetime: createdAt,
	})
}

func (m *MessageService) Create(ctx context.Context, roomID string, userID string, content string) (repository.Message, error) {
	err := checkValidRequest(roomID, userID)
	if err != nil {
		return repository.Message{}, err
	}

	return m.q.CreateMessage(ctx, repository.CreateMessageParams{
		RoomID:  roomID,
		UserID:  userID,
		ID:      ulid.Make().String(),
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
