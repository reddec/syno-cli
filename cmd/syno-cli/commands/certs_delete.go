package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"text/tabwriter"

	"github.com/reddec/syno-cli/pkg/client"
)

//nolint:staticcheck
type CertsDelete struct {
	SynoClient `group:"Synology Client" namespace:"synology" env-namespace:"SYNOLOGY"`
	Format     string `short:"f" long:"format" env:"FORMAT" description:"Output format" default:"table" choice:"table" choice:"json"`
	Args       struct {
		ID string `positional-arg-name:"id" env:"NAME" description:"certificate ID or name" required:"true"`
	} `positional-args:"true"`
}

func (lc *CertsDelete) Execute([]string) error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer cancel()

	syno := lc.Client()

	list, err := syno.ListCerts(ctx)
	if err != nil {
		return err
	}

	var certID string
	for _, c := range list {
		if c.ID == lc.Args.ID {
			certID = c.ID
			break
		} else if c.Description == lc.Args.ID {
			certID = c.ID
		}
	}

	if certID == "" {
		return fmt.Errorf("unknown name or id") //nolint:goerr113
	}

	info, err := syno.DeleteCertByID(ctx, certID)
	if err != nil {
		return err
	}

	return lc.show(certID, info)
}

//nolint:gomnd
func (lc *CertsDelete) show(id string, info *client.ServerStatus) error {
	switch lc.Format {
	case fmtJSON:
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(info)
	case fmtTable:
		fallthrough
	default:
		tw := tabwriter.NewWriter(os.Stdout, 3, 4, 2, ' ', 0)
		_, _ = fmt.Fprintln(tw,
			"ID", "\t",
			"Server restarted", "\t",
		)
		_, _ = fmt.Fprintln(tw,
			id, "\t",
			info.ServerRestarted, "\t",
		)
		return tw.Flush()
	}
}
