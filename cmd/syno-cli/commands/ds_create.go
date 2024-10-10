package commands

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"os/signal"

	"github.com/reddec/syno-cli/pkg/client"
)

type DsCreate struct {
	Logging
	SynoClient  `group:"Synology Client" namespace:"synology" env-namespace:"SYNOLOGY"`
	Format      client.FileType `short:"f" long:"format" env:"FORMAT" description:"File format" default:"auto" choice:"torrent" choice:"txt" choice:"nzb" choice:"auto"`
	Destination string          `short:"d" long:"destination" env:"DESTINATION" description:"Destination directory" default:"Downloads"`
	Args        struct {
		Ref string `positional-arg-name:"ref" description:"URL or file name. If not set or set to - (dash) - STDIN will be used"`
	} `positional-args:"yes"`
}

func (cmd *DsCreate) Execute([]string) error {
	cmd.SetupLogging()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer cancel()

	params := client.DownloadTask{
		FileType:    cmd.Format,
		Destination: cmd.Destination,
	}

	syno := cmd.Client()

	if u, err := url.Parse(cmd.Args.Ref); err == nil && u.Scheme != "" {
		slog.Debug("ref is URL", "url", u.Redacted())
		params.URL = []string{cmd.Args.Ref}
	} else if cmd.Args.Ref == "" || cmd.Args.Ref == "-" {
		slog.Debug("ref is STDIN payload")
		params.File = os.Stdin
	} else {
		slog.Debug("ref is file")
		f, err := os.Open(cmd.Args.Ref)
		if err != nil {
			return fmt.Errorf("open file: %w", err)
		}
		defer f.Close()
		params.File = f
	}
	slog.Debug("creating download task", "destination", params.Destination)
	err := syno.DownloadStation().Create(ctx, params)
	if err != nil {
		return err
	}
	slog.Info("created task in Download Station")
	return nil
}
