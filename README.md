# Synology CLI

Unofficial wrapper over Synology API in Go.

Focus on administrative tasks.

Supports:

* X86-64 and ARM [builds](https://github.com/reddec/syno-cli/releases/latest)
* Universal docker image (arm and amd64)
* As [CLI](https://github.com/reddec/syno-cli/releases/latest) and as [library](https://pkg.go.dev/github.com/reddec/syno-cli)

## Installation

* Pre-built binaries from [releases](https://github.com/reddec/syno-cli/releases/latest)
* Docker (universal): `ghcr.io/reddec/syno-cli:<release>` (see [releases](https://github.com/reddec/syno-cli/releases/latest))
* From source (requires latest Go): `go install github.com/reddec/syno-cli/cmd/syno-cli@latest`

## Usage

Each command supports `--help` option.


### main

```
Usage:
  syno-cli [OPTIONS] <cert>

Unofficial CLI for Synology DSM
syno-cli 0.0.1, commit none, built at 2022-01-10T14:02:11Z by goreleaser
Author: Aleksandr Baryshnikov <owner@reddec.net>

Help Options:
  -h, --help  Show this help message

Available commands:
  cert  manager certificates (aliases: certificates, certificate, certs, cert, c)
```

### cert

```
Usage:
  syno-cli [OPTIONS] cert <command>

Help Options:
  -h, --help      Show this help message

Available commands:
  auto    automatically issue and push certificates (aliases: dns01, lego, a)
  delete  delete certificate (aliases: remove, rm, del, d)
  list    list certificates (aliases: ls, l)
  upload  upload certificate (aliases: up, u)
```

### automatic certs


```
Usage:
  syno-cli [OPTIONS] cert auto [auto-OPTIONS] [domain...]

Help Options:
  -h, --help                   Show this help message

[auto command options]
      -c, --cache-dir=         Cache location for accounts information (default: .cache) [$CACHE_DIR]
      -r, --renew-before=      Renew certificate time reserve (default: 720h) [$RENEW_BEFORE]
      -e, --email=             Email for contact [$EMAIL]
      -p, --provider=          DNS challenge provider [$PROVIDER]
      -D, --dns=               Custom resolvers (default: 8.8.8.8) [$DNS]
      -t, --timeout=           DNS challenge timeout (default: 1m) [$TIMEOUT]
      -d, --domains=           Domains names to issue [$DOMAINS]

    Synology Client:
          --synology.user=     Synology username [$SYNOLOGY_USER]
          --synology.password= Synology password [$SYNOLOGY_PASSWORD]
          --synology.url=      Synology URL (default: http://localhost:5000) [$SYNOLOGY_URL]
```