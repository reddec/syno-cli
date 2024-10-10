package commands

import (
	"crypto/tls"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"os"
	"time"

	"github.com/reddec/syno-cli/pkg/client"
)

type SynoClient struct {
	User     string        `long:"user" env:"USER" description:"Synology username" required:"true"`
	Password string        `long:"password" env:"PASSWORD" description:"Synology password" required:"true"`
	URL      string        `long:"url" env:"URL" description:"Synology URL" default:"http://localhost:5000"`
	Insecure bool          `long:"insecure" env:"INSECURE" description:"Disable TLS (HTTPS) verification"`
	Timeout  time.Duration `long:"timeout" env:"TIMEOUT" description:"Default timeout" default:"30s"`
}

func (sc SynoClient) Client() *client.Client {
	jar, err := cookiejar.New(nil)
	if err != nil {
		panic(err) // impossible
	}
	var httpClient = &http.Client{
		Jar:     jar,
		Timeout: sc.Timeout,
	}
	if sc.Insecure {
		httpClient.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	}
	return client.New(client.Config{
		Client:   httpClient,
		User:     sc.User,
		Password: sc.Password,
		URL:      sc.URL,
	})
}

const (
	fmtJSON  = "json"
	fmtTable = "table"
)

type Logging struct {
	Debug bool `long:"debug" env:"DEBUG" description:"Enable debug logging"`
}

func (l *Logging) SetupLogging() {
	if !l.Debug {
		return
	}
	lvl := new(slog.LevelVar)
	lvl.Set(slog.LevelDebug)

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: lvl,
	}))
	slog.SetDefault(logger)
}
