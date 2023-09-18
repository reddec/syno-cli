package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/signal"
	"text/tabwriter"

	"github.com/reddec/syno-cli/pkg/client"
)

//nolint:staticcheck
type CertsUpload struct {
	SynoClient `group:"Synology Client" namespace:"synology" env-namespace:"SYNOLOGY"`
	Key        string `short:"k" long:"key" env:"KEY" description:"Path to private key. Use - (dash) to read it from stdin" default:"-"`
	Cert       string `short:"c" long:"cert" env:"CERT" description:"Path to server certificate" required:"true"`
	CA         string `short:"C" long:"ca" env:"CA" description:"Path to intermediate certificate"`
	Format     string `short:"f" long:"format" env:"FORMAT" description:"Output format" default:"table" choice:"table" choice:"json"`
	Default    bool   `short:"d" long:"default" env:"DEFAULT" description:"Set certificate as default"`
	Args       struct {
		Name string `positional-arg-name:"name" env:"NAME" description:"certificate name" required:"true"`
	} `positional-args:"true"`
}

func (lc *CertsUpload) Execute([]string) error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer cancel()

	syno := lc.Client()

	var privateFile io.ReadCloser
	if lc.Key == "-" {
		privateFile = os.Stdin
	} else if f, err := os.Open(lc.Key); err == nil {
		privateFile = f
	} else {
		return err
	}
	defer privateFile.Close()

	certFile, err := os.Open(lc.Cert)
	if err != nil {
		return err
	}
	defer certFile.Close()

	var caFile io.Reader
	if lc.CA != "" {
		f, err := os.Open(lc.CA)
		if err != nil {
			return err
		}
		caFile = f
		defer f.Close()
	}

	info, err := syno.UploadCert(ctx, client.NewCertificate{
		Name:      lc.Args.Name,
		AsDefault: lc.Default,
		Cert:      certFile,
		CA:        caFile,
		Key:       privateFile,
	})
	if err != nil {
		return err
	}

	return lc.show(info)
}

//nolint:gomnd
func (lc *CertsUpload) show(info *client.CertUploadResult) error {
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
			info.CertificateID, "\t",
			info.ServerRestarted, "\t",
		)
		return tw.Flush()
	}
}
