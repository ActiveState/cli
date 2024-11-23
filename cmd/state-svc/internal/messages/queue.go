package messages

import (
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/graph"
)

type Queue struct {
	queue map[string]map[string]*graph.Message
}

func NewQueue() *Queue {
	return &Queue{
		queue: make(map[string]map[string]*graph.Message),
	}
}

func (q *Queue) Queue(topic string, message string) error {
	if _, ok := q.queue[topic]; !ok {
		q.queue[topic] = make(map[string]*graph.Message)
	}
	msg := NewMessage(topic, message)
	q.queue[topic][msg.ID] = msg
	return nil
}

func (q *Queue) Messages() ([]*graph.Message, error) {
	var messages []*graph.Message
	for _, topic := range q.queue {
		for _, message := range topic {
			messages = append(messages, message)
		}
	}
	return messages, nil
}

func (q *Queue) Dequeue(messageIDs []string) error {
	for _, messageID := range messageIDs {
		err := q.dequeueMessages(messageID)
		if err != nil {
			return errs.Wrap(err, "failed to dequeue message")
		}
	}
	return nil
}

func (q *Queue) dequeueMessages(messageID string) error {
	for _, topic := range q.queue {
		for _, msg := range topic {
			if msg.ID != messageID {
				continue
			}
			delete(topic, messageID)
			return nil
		}
	}
	return nil
}

func (q *Queue) Close() error {
	return nil
}
