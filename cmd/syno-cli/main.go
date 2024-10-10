package main

import (
	"fmt"
	"os"

	"github.com/jessevdk/go-flags"

	"github.com/reddec/syno-cli/cmd/syno-cli/commands"
)

//nolint:gochecknoglobals
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
	builtBy = "unknown"
)

//nolint:staticcheck
type Config struct {
	Cert struct {
		List   commands.CertsList   `command:"list" description:"list certificates" alias:"ls" alias:"l"`
		Upload commands.CertsUpload `command:"upload" description:"upload certificate" alias:"up" alias:"u"`
		Delete commands.CertsDelete `command:"delete" description:"delete certificate" alias:"remove" alias:"rm"  alias:"del" alias:"d"`
		Auto   commands.CertsAuto   `command:"auto" description:"automatically issue and push certificates" alias:"dns01" alias:"lego" alias:"a"`
	} `command:"cert" description:"manager certificates" alias:"certificates" alias:"certificate" alias:"certs" alias:"cert" alias:"c"`
	DS struct {
		Create commands.DsCreate `command:"create" description:"create task" alias:"add" alias:"new" alias:"c"`
		List   commands.DsList   `command:"list" description:"list tasks" alias:"ls" alias:"l"`
	} `command:"ds" description:"download station" alias:"download-station" alias:"download" alias:"dl" alias:"d"`
}

func main() {
	var config Config
	parser := flags.NewParser(&config, flags.Default)
	parser.ShortDescription = "Synology CLI"
	parser.LongDescription = fmt.Sprintf("Unofficial CLI for Synology DSM\nsyno-cli %s, commit %s, built at %s by %s\nAuthor: Aleksandr Baryshnikov <owner@reddec.net>", version, commit, date, builtBy)

	if _, err := parser.Parse(); err != nil {
		os.Exit(1)
	}
}
