package buildlog

import (
	"encoding/json"
	"time"

	"github.com/ActiveState/cli/pkg/platform/runtime/artifact"
)

type MessageEnum int

const (
	BuildSucceeded MessageEnum = iota
	BuildFailed
	ArtifactStarted
	ArtifactSucceeded
	ArtifactFailed
	ArtifactProgress
	UnknownMessage
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
	default:
		return UnknownMessage
	}
}

type buildMessage struct {
	BaseMessage
	RecipeID  string    `json:"recipe_id"`
	Timestamp time.Time `json:"timestamp"`
	CacheHit  bool      `json:"cache_hit"`
}

type buildFailedMessage struct {
	buildMessage
	ErrorMessage string `json:"error_message"`
}

type artifactMessage struct {
	BaseMessage
	RecipeID   string              `json:"recipe_id"`
	ArtifactID artifact.ArtifactID `json:"artifact_id"`
	Timestamp  time.Time           `json:"timestamp"`
	// CacheHit indicates if an artifact has been originally built for a different recipe
	CacheHit bool `json:"cache_hit"`
}

type artifactSucceededMessage struct {
	artifactMessage
	ArtifactURI      string `json:"artifact_uri"`
	ArtifactChecksum string `json:"artifact_checksum"`
	ArtifactMIMEType string `json:"artifact_mime_type"`
	LogURI           string `json:"log_uri"`
}

type artifactFailedMessage struct {
	artifactMessage
	ErrorMessage string `json:"error_message"`
	LogURI       string `json:"log_uri"`
}

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
		var bm buildMessage
		err := json.Unmarshal(b, &bm)
		return bm, err
	case BuildFailed:
		var fm buildFailedMessage
		err := json.Unmarshal(b, &fm)
		return fm, err
	case ArtifactStarted:
		var am artifactMessage
		err := json.Unmarshal(b, &am)
		return am, err
	case ArtifactSucceeded:
		var am artifactSucceededMessage
		err := json.Unmarshal(b, &am)
		return am, err
	case ArtifactFailed:
		var am artifactFailedMessage
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
