package commands

import (
	"crypto/tls"
	"github.com/reddec/syno-cli/pkg/client"
	"net/http"
)

type SynoClient struct {
	User     string `long:"user" env:"USER" description:"Synology username" required:"true"`
	Password string `long:"password" env:"PASSWORD" description:"Synology password" required:"true"`
	URL      string `long:"url" env:"URL" description:"Synology URL" default:"http://localhost:5000"`
	Insecure bool   `long:"insecure" env:"INSECURE" description:"Disable TLS (HTTPS) verification"`
}

func (sc SynoClient) Client() *client.Client {
	var httpClient = http.DefaultClient
	if sc.Insecure {
		httpClient = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
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
