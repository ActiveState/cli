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

type message struct {
	messager
}

type baseMessage struct {
	Type string `json:"type"`
}

func (bm baseMessage) MessageType() MessageEnum {
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
	baseMessage
	RecipeID  string    `json:"recipe_id"`
	Timestamp time.Time `json:"timestamp"`
	CacheHit  bool      `json:"cache_hit"`
}

type buildFailedMessage struct {
	buildMessage
	ErrorMessage string `json:"error_message"`
}

type artifactMessage struct {
	baseMessage
	RecipeID   string              `json:"recipe_id"`
	ArtifactID artifact.ArtifactID `json:"artifact_id"`
	Timestamp  time.Time           `json:"timestamp"`
	CacheHit   bool                `json:"cache_hit"`
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

type artifactProgressMessage struct {
	baseMessage
	ArtifactID artifact.ArtifactID  `json:"artifact_id"`
	Timestamp  time.Time            `json:"timestamp"`
	Source     string               `json:"source"`
	PipeName   string               `json:"pipe_name"`
	Body       artifactProgressBody `json:"body"`
}

type artifactProgressBody struct {
	Facility string `json:"facility"`
	Message  string `json:"msg"`
}

func unmarshalSpecialMessage(baseMsg baseMessage, b []byte) (messager, error) {
	switch baseMsg.MessageType() {
	case BuildSucceeded:
		var bm buildMessage
		if err := json.Unmarshal(b, bm); err != nil {
			return bm, err
		}
	case BuildFailed:
		var fm buildFailedMessage
		if err := json.Unmarshal(b, &fm); err != nil {
			return fm, err
		}
	case ArtifactStarted:
		var am artifactMessage
		if err := json.Unmarshal(b, &am); err != nil {
			return am, err
		}
	case ArtifactSucceeded:
		var am artifactSucceededMessage
		if err := json.Unmarshal(b, &am); err != nil {
			return am, err
		}
	case ArtifactFailed:
		var am artifactFailedMessage
		if err := json.Unmarshal(b, &am); err != nil {
			return am, err
		}
	case ArtifactProgress:
		var am artifactProgressMessage
		if err := json.Unmarshal(b, &am); err != nil {
			return am, err
		}
	}
	return baseMsg, nil
}

func UnmarshalJSON(b []byte) (messager, error) {
	var bm baseMessage
	if err := json.Unmarshal(b, &bm); err != nil {
		return nil, err
	}

	return unmarshalSpecialMessage(bm, b)
}

func (m *message) UnmarshalJSON(b []byte) error {
	mm, err := UnmarshalJSON(b)
	if err != nil {
		return err
	}
	m = &message{mm}
	return nil
}
