package client_test

import (
	"context"
	"testing"
	"time"

	"github.com/reddec/syno-cli/pkg/client"
	"github.com/stretchr/testify/require"
)

func TestClient_Download(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	syno := client.New(client.FromEnv(environ()))
	err := syno.Download(ctx, "Downloads", `https://releases.ubuntu.com/22.04/ubuntu-22.04-live-server-amd64.iso.torrent`)
	require.NoError(t, err)
}
