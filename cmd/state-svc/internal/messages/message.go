package messages

import (
	"github.com/ActiveState/cli/internal/graph"
	"github.com/google/uuid"
)

func NewMessage(topic string, message string) *graph.Message {
	return &graph.Message{
		ID:      uuid.New().String(),
		Topic:   topic,
		Message: message,
	}
}
