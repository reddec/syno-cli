package client_test

import (
	"context"
	"log"
	"testing"
	"time"

	"github.com/reddec/syno-cli/pkg/client"
	"github.com/stretchr/testify/require"
)

func TestClient_Download(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	syno := client.New(client.FromEnv(environ()))
	err := syno.DownloadStation().Download(ctx, "Downloads", `https://releases.ubuntu.com/22.04/ubuntu-22.04-live-server-amd64.iso.torrent`)
	require.NoError(t, err)
}

func TestDownloadStation_List(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	syno := client.New(client.FromEnv(environ()))
	list, err := syno.DownloadStation().List(ctx, 0, -1)
	require.NoError(t, err)
	require.NotEmpty(t, list)
	log.Printf("%+v", *list)
}
