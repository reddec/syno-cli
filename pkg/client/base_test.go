package client_test

import (
	"bufio"
	"context"
	"log"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/reddec/syno-cli/pkg/client"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_APIVersion(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	syno := client.New(client.FromEnv(environ()))
	info, err := syno.APIVersion(ctx, "SYNO.API.Info")
	require.NoError(t, err)
	assert.NotEmpty(t, info.Path)
	assert.NotEmpty(t, info.MaxVersion)
}

func TestClient_Login(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	syno := client.New(client.FromEnv(environ()))
	err := syno.Login(ctx)
	require.NoError(t, err)
}

func environ() func(string) string {
	var env = make(map[string]string)
	for _, kv := range os.Environ() {
		parts := strings.SplitN(kv, "=", 2)
		env[parts[0]] = parts[1]
	}

	f, err := os.Open(".env")
	if err != nil {
		log.Println("env not loaded:", err)
		return os.Getenv
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "#") {
			continue
		}
		kv := strings.SplitN(line, "=", 2)
		env[kv[0]] = kv[1]
	}

	return func(s string) string {
		return env[s]
	}
}
