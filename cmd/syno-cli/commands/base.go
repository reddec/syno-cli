package commands

import (
	"github.com/reddec/syno-cli/pkg/client"
)

type SynoClient struct {
	User     string `long:"user" env:"USER" description:"Synology username" required:"true"`
	Password string `long:"password" env:"PASSWORD" description:"Synology password" required:"true"`
	URL      string `long:"url" env:"URL" description:"Synology URL" default:"http://localhost:5000"`
}

func (sc SynoClient) Client() *client.Client {
	return client.New(client.Config{
		User:     sc.User,
		Password: sc.Password,
		URL:      sc.URL,
	})
}

const (
	fmtJSON  = "json"
	fmtTable = "table"
)
