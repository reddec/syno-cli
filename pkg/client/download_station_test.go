package client_test

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/reddec/syno-cli/pkg/client"
)

func TestClient_Download(t *testing.T) {
	lvl := new(slog.LevelVar)
	lvl.Set(slog.LevelDebug)

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: lvl,
	}))
	slog.SetDefault(logger)

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	syno := client.New(client.FromEnv(environ()))
	err := syno.DownloadStation().Download(ctx, "Downloads", `https://webtorrent.io/torrents/cosmos-laundromat.torrent`)
	require.NoError(t, err)
}

func TestClient_TorrentFile(t *testing.T) {
	lvl := new(slog.LevelVar)
	lvl.Set(slog.LevelDebug)

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: lvl,
	}))
	slog.SetDefault(logger)

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, `https://webtorrent.io/torrents/cosmos-laundromat.torrent`, nil)
	require.NoError(t, err)
	res, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, res.StatusCode)
	defer res.Body.Close()

	torrentFile, err := io.ReadAll(res.Body)
	require.NoError(t, err)

	syno := client.New(client.FromEnv(environ()))
	err = syno.DownloadStation().Create(ctx,
		client.DownloadTask{
			File:        bytes.NewReader(torrentFile),
			Destination: "Downloads",
		})
	require.NoError(t, err)
}

func TestDownloadStation_List(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	syno := client.New(client.FromEnv(environ()))
	list, err := syno.DownloadStation().List(ctx, 0, -1)
	require.NoError(t, err)
	require.NotEmpty(t, list)

	t.Logf("%+v", *list)
}
