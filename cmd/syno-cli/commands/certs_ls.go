package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/reddec/syno-cli/pkg/client"
)

//nolint:staticcheck
type CertsList struct {
	SynoClient `group:"Synology Client" namespace:"synology" env-namespace:"SYNOLOGY"`
	Format     string `short:"f" long:"format" env:"FORMAT" description:"How to show output" default:"table" choice:"table" choice:"json"`
}

func (lc *CertsList) Execute([]string) error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer cancel()

	syno := lc.Client()
	list, err := syno.ListCerts(ctx)
	if err != nil {
		return err
	}

	return lc.show(list)
}

//nolint:gomnd
func (lc *CertsList) show(list []client.Certificate) error {
	switch lc.Format {
	case fmtJSON:
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(list)
	case fmtTable:
		fallthrough
	default:
		tw := tabwriter.NewWriter(os.Stdout, 3, 4, 2, ' ', 0)
		_, _ = fmt.Fprintln(tw,
			"Status", "\t",
			"ID", "\t",
			"Name", "\t",
			"SAN", "\t",
			"Issuer", "\t",
			"Since", "\t",
			"Expired", "\t",
		)
		for _, item := range list {
			sign := ""
			if item.IsDefault {
				sign += "*"
			}
			if item.IsBroken {
				sign += "!"
			}
			if item.Expired() {
				sign += "-"
			}
			_, _ = fmt.Fprintln(tw,
				sign, "\t",
				item.ID, "\t",
				item.Description, "\t",
				strings.Join(item.Subject.SubAltName, ","), "\t",
				item.Issuer.CommonName, "\t",
				item.ValidFrom.Time().Format(time.RFC822), "\t",
				item.ValidTill.Time().Format(time.RFC822),
			)
		}
		return tw.Flush()
	}
}
