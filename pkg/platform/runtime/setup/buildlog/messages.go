package buildlog

import (
	"encoding/json"
	"time"

	"github.com/ActiveState/cli/pkg/platform/runtime/artifact"
)

// MessageEnum enumerates the build events that can be expected on the buildlog websocket connection
type MessageEnum int

const (
	UnknownMessage MessageEnum = iota
	BuildStarted
	BuildSucceeded
	BuildFailed
	ArtifactStarted
	ArtifactSucceeded
	ArtifactFailed
	ArtifactProgress
	Heartbeat
)

type messager interface {
	MessageType() MessageEnum
}

type Message struct {
	messager
}

type BaseMessage struct {
	Type string `json:"type"`
}

func (bm BaseMessage) MessageType() MessageEnum {
	switch bm.Type {
	case "build_started":
		return BuildStarted
	case "build_succeeded":
		return BuildSucceeded
	case "build_failed":
		return BuildFailed
	case "artifact_started":
		return ArtifactStarted
	case "artifact_succeeded":
		return ArtifactSucceeded
	case "artifact_failed":
		return ArtifactFailed
	case "artifact_progress":
		return ArtifactProgress
	case "heartbeat":
		return Heartbeat
	default:
		return UnknownMessage
	}
}

// BuildMessage comprises status information about a build
type BuildMessage struct {
	BaseMessage
	RecipeID  string    `json:"recipe_id"`
	Timestamp time.Time `json:"timestamp"`
}

// BuildFailedMessage extends a BuildMessage with an error message
type BuildFailedMessage struct {
	BuildMessage
	ErrorMessage string `json:"error_message"`
}

// ArtifactMessage holds status information for an individual artifact
type ArtifactMessage struct {
	BaseMessage
	RecipeID   string              `json:"recipe_id"`
	ArtifactID artifact.ArtifactID `json:"artifact_id"`
	Timestamp  time.Time           `json:"timestamp"`
	// CacheHit indicates if an artifact has been originally built for a different recipe
	CacheHit bool `json:"cache_hit"`
}

// ArtifactSucceededMessage extends an ArtifactMessage with information that becomes available after an artifact built successfully
type ArtifactSucceededMessage struct {
	ArtifactMessage
	ArtifactURI      string `json:"artifact_uri"`
	ArtifactChecksum string `json:"artifact_checksum"`
	ArtifactMIMEType string `json:"artifact_mime_type"`
	LogURI           string `json:"log_uri"`
}

// ArtifactFailedMessage extends an ArtifactMessage with error information
type ArtifactFailedMessage struct {
	ArtifactMessage
	ErrorMessage string `json:"error_message"`
	LogURI       string `json:"log_uri"`
}

// ArtifactProgressMessage forwards detailed logging information send for an artifact
type ArtifactProgressMessage struct {
	BaseMessage
	ArtifactID artifact.ArtifactID  `json:"artifact_id"`
	Timestamp  string               `json:"timestamp"`
	Source     string               `json:"source"`
	PipeName   string               `json:"pipe_name"`
	Body       ArtifactProgressBody `json:"body"`
}

type ArtifactProgressBody struct {
	Facility string `json:"facility"`
	Message  string `json:"msg"`
}

func unmarshalSpecialMessage(baseMsg BaseMessage, b []byte) (messager, error) {
	switch baseMsg.MessageType() {
	case BuildSucceeded:
		var bm BuildMessage
		err := json.Unmarshal(b, &bm)
		return bm, err
	case BuildFailed:
		var fm BuildFailedMessage
		err := json.Unmarshal(b, &fm)
		return fm, err
	case Heartbeat:
		var bm BuildMessage
		err := json.Unmarshal(b, &bm)
		return bm, err
	case ArtifactStarted:
		var am ArtifactMessage
		err := json.Unmarshal(b, &am)
		return am, err
	case ArtifactSucceeded:
		var am ArtifactSucceededMessage
		err := json.Unmarshal(b, &am)
		return am, err
	case ArtifactFailed:
		var am ArtifactFailedMessage
		err := json.Unmarshal(b, &am)
		return am, err
	case ArtifactProgress:
		var am ArtifactProgressMessage
		err := json.Unmarshal(b, &am)
		return am, err
	}
	return baseMsg, nil
}

func UnmarshalJSON(b []byte) (messager, error) {
	var bm BaseMessage
	if err := json.Unmarshal(b, &bm); err != nil {
		return nil, err
	}

	return unmarshalSpecialMessage(bm, b)
}

func (m *Message) UnmarshalJSON(b []byte) error {
	mm, err := UnmarshalJSON(b)
	if err != nil {
		return err
	}
	*m = Message{mm}
	return nil
}
