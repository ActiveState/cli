package buildlog

import (
	"encoding/json"
	"testing"

	"github.com/ActiveState/cli/pkg/platform/runtime/artifact"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestArtifactProgressMessage(t *testing.T) {
	prgMsg := `{"body": {"facility": "INFO", "msg": "message"}, "artifact_id": "00000000-0000-0000-0000-000000000001", "timestamp": "2021-06-24T21:27:31.487131", "type": "artifact_progress", "source": "builder", "pipe_name": "stdout"}`

	var msg Message
	err := json.Unmarshal([]byte(prgMsg), &msg)
	require.NoError(t, err)

	assert.Equal(t, ArtifactProgress, msg.MessageType())
	apm, ok := msg.messager.(ArtifactProgressMessage)
	require.True(t, ok)
	assert.Equal(t, "artifact_progress", apm.Type)
	assert.Equal(t, "message", apm.Body.Message)
	assert.Equal(t, "INFO", apm.Body.Facility)
	assert.Equal(t, artifact.ArtifactID("00000000-0000-0000-0000-000000000001"), apm.ArtifactID)
	assert.Equal(t, "2021-06-24T21:27:31.487131", apm.Timestamp)
}
