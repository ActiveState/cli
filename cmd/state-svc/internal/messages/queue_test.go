package messages

import (
	"testing"

	"github.com/ActiveState/cli/internal/graph"
	"github.com/stretchr/testify/assert"
)

func TestNewQueue(t *testing.T) {
	q := NewQueue()
	assert.NotNil(t, q)
	assert.Empty(t, q.queue)
}

func TestQueue_Queue(t *testing.T) {
	tests := []struct {
		name     string
		messages []graph.Message
		want     map[string]int // topic -> expected message count
	}{
		{
			name: "queue first message",
			messages: []graph.Message{
				{Topic: "topic1", Message: "message1"},
			},
			want: map[string]int{"topic1": 1},
		},
		{
			name: "queue second message in same topic",
			messages: []graph.Message{
				{Topic: "topic1", Message: "message1"},
				{Topic: "topic1", Message: "message2"},
			},
			want: map[string]int{"topic1": 2},
		},
		{
			name: "queue message in different topic",
			messages: []graph.Message{
				{Topic: "topic1", Message: "message1"},
				{Topic: "topic2", Message: "message3"},
			},
			want: map[string]int{"topic1": 1, "topic2": 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := NewQueue()

			for _, m := range tt.messages {
				err := q.Queue(m.Topic, m.Message)
				assert.NoError(t, err)
			}

			for topic, count := range tt.want {
				assert.Len(t, q.queue[topic], count)
			}
		})
	}
}

func TestQueue_Messages(t *testing.T) {
	tests := []struct {
		name      string
		messages  []graph.Message
		wantCount int
	}{
		{
			name:      "empty queue",
			messages:  nil,
			wantCount: 0,
		},
		{
			name: "single message",
			messages: []graph.Message{
				{Topic: "topic1", Message: "message1"},
			},
			wantCount: 1,
		},
		{
			name: "multiple messages across topics",
			messages: []graph.Message{
				{Topic: "topic1", Message: "message1"},
				{Topic: "topic1", Message: "message2"},
				{Topic: "topic2", Message: "message3"},
			},
			wantCount: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := NewQueue()

			for _, m := range tt.messages {
				err := q.Queue(m.Topic, m.Message)
				assert.NoError(t, err)
			}

			msgs, err := q.Messages()
			assert.NoError(t, err)
			assert.Len(t, msgs, tt.wantCount)
		})
	}
}

func TestQueue_Dequeue(t *testing.T) {
	tests := []struct {
		name          string
		messages      []graph.Message
		dequeueIDs    []string
		wantRemaining int
	}{
		{
			name: "dequeue single message",
			messages: []graph.Message{
				{Topic: "topic1", Message: "message1"},
			},
			dequeueIDs:    nil, // Will be populated during test with actual message ID
			wantRemaining: 0,
		},
		{
			name: "dequeue multiple messages",
			messages: []graph.Message{
				{Topic: "topic1", Message: "message1"},
				{Topic: "topic2", Message: "message2"},
			},
			dequeueIDs:    nil, // Will be populated during test with actual message IDs
			wantRemaining: 0,
		},
		{
			name: "dequeue non-existent message",
			messages: []graph.Message{
				{Topic: "topic1", Message: "message1"},
			},
			dequeueIDs:    []string{"non-existent-id"},
			wantRemaining: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := NewQueue()

			for _, m := range tt.messages {
				err := q.Queue(m.Topic, m.Message)
				assert.NoError(t, err)
			}

			if tt.dequeueIDs == nil {
				msgs, err := q.Messages()
				assert.NoError(t, err)

				tt.dequeueIDs = make([]string, len(msgs))
				for i, msg := range msgs {
					tt.dequeueIDs[i] = msg.ID
				}
			}

			err := q.Dequeue(tt.dequeueIDs)
			assert.NoError(t, err)

			remaining, err := q.Messages()
			assert.NoError(t, err)
			assert.Len(t, remaining, tt.wantRemaining)
		})
	}
}
