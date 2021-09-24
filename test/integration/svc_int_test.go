package integration

import (
	"testing"
	"time"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/svcmanager"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"
)

func TestSvcManagerAndModel(t *testing.T) {
	cfg, err := config.New()
	require.NoError(t, err)
	svcManager := svcmanager.New(cfg)
	require.NoError(t, svcManager.Start())

	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	require.NoError(t, svcManager.WaitWithContext(ctx))

	require.NoError(t, err)
	require.NoError(t, model.StopServer(cfg))
}
