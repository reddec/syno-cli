package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"text/tabwriter"
	"time"

	"github.com/reddec/syno-cli/pkg/client"
)

type DsList struct {
	Logging
	SynoClient `group:"Synology Client" namespace:"synology" env-namespace:"SYNOLOGY"`
	Format     string `short:"f" long:"format" env:"FORMAT" description:"How to show output" default:"table" choice:"table" choice:"json"`
	Offset     int    `short:"o" long:"offset" env:"OFFSET" description:"Offset" default:"0"`
	Limit      int    `short:"l" long:"limit" env:"LIMIT" description:"Max number of items" default:"1000"`
}

func (cmd *DsList) Execute([]string) error {
	cmd.SetupLogging()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer cancel()

	syno := cmd.Client()
	info, err := syno.DownloadStation().List(ctx, cmd.Offset, cmd.Limit)
	if err != nil {
		return fmt.Errorf("list tasks: %w", err)
	}
	return cmd.show(info.Tasks)
}

//nolint:gomnd
func (cmd *DsList) show(list []client.ScheduledTask) error {
	switch cmd.Format {
	case fmtJSON:
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(list)
	case fmtTable:
		fallthrough
	default:
		tw := tabwriter.NewWriter(os.Stdout, 3, 4, 2, ' ', 0)
		_, _ = fmt.Fprintln(tw,
			"ID", "\t",
			"User", "\t",
			"Status", "\t",
			"Type", "\t",
			"Size", "\t",
			"Created", "\t",
			"Title", "\t",
		)
		for _, item := range list {
			_, _ = fmt.Fprintln(tw,
				item.ID, "\t",
				item.Username, "\t",
				item.Status, "\t",
				item.Type, "\t",
				item.Size, "\t",
				time.Unix(item.Additional.Detail.CreateTime, 0).Format(time.RFC3339), "\t",
				item.Title,
			)
		}
		return tw.Flush()
	}
}
