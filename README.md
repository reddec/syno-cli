# Synology CLI

Unofficial wrapper over Synology API in Go.

Focus on administrative tasks.

* Tutorial for [automatic SSL certificates on NAS](https://reddec.net/articles/how-to-get-ssl-on-synology/)

> [!TIP]
> It does support creating tasks in Download Station using files (torrent, nzb, urls); though it uses undocumented API.


Supports:

* X86-64 and ARM [builds](https://github.com/reddec/syno-cli/releases/latest)
* Universal docker image (arm and amd64)
* As [CLI](https://github.com/reddec/syno-cli/releases/latest) and
  as [library](https://pkg.go.dev/github.com/reddec/syno-cli)

## Development notes

- Stable are only tags prefixed by `v` (ex: `v0.1.4`)

## Installation

* Pre-built binaries from [releases](https://github.com/reddec/syno-cli/releases/latest)
* Docker (universal): `ghcr.io/reddec/syno-cli:<release>` (
  see [releases](https://github.com/reddec/syno-cli/releases/latest))
* From source (requires latest Go): `go install github.com/reddec/syno-cli/cmd/syno-cli@latest`

## Usage

Each command supports `--help` option.

### main

```
Usage:
  syno-cli [OPTIONS] <cert>

Unofficial CLI for Synology DSM
Author: Aleksandr Baryshnikov <owner@reddec.net>

Help Options:
  -h, --help  Show this help message

Available commands:
  cert  manager certificates (aliases: certificates, certificate, certs, cert, c)
  ds    download station (aliases: download-station, download, dl, d)
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
          --synology.insecure  Disable TLS (HTTPS) verification [$SYNOLOGY_INSECURE]
```

## Download station

In progress. Already supports creating task from files.

API supported:

- create task
- list tasks

Command: `syno-cli ds ...`

```
Usage:
  syno-cli [OPTIONS] ds <create | list>

Help Options:
  -h, --help      Show this help message

Available commands:
  create  create task (aliases: add, new, c)
  list    list tasks (aliases: ls, l)
```

### Create download task

```
Usage:
  syno-cli [OPTIONS] ds create [create-OPTIONS] [ref]

Help Options:
  -h, --help                              Show this help message

[create command options]
          --debug                         Enable debug logging [$DEBUG]
      -f, --format=[torrent|txt|nzb|auto] File format (default: auto) [$FORMAT]
      -d, --destination=                  Destination directory (default: Downloads) [$DESTINATION]

    Synology Client:
          --synology.user=                Synology username [$SYNOLOGY_USER]
          --synology.password=            Synology password [$SYNOLOGY_PASSWORD]
          --synology.url=                 Synology URL (default: http://localhost:5000) [$SYNOLOGY_URL]
          --synology.insecure             Disable TLS (HTTPS) verification [$SYNOLOGY_INSECURE]
          --synology.timeout=             Default timeout (default: 30s) [$SYNOLOGY_TIMEOUT]

[create command arguments]
  ref:                                    URL or file name. If not set or set to - (dash) - STDIN will be used
```

- If `ref` is set it could be URL, including magnet or path to file.

### List tasks

```
Usage:
  syno-cli [OPTIONS] ds list [list-OPTIONS]

Help Options:
  -h, --help                    Show this help message

[list command options]
          --debug               Enable debug logging [$DEBUG]
      -f, --format=[table|json] How to show output (default: table) [$FORMAT]
      -o, --offset=             Offset (default: 0) [$OFFSET]
      -l, --limit=              Max number of items (default: 1000) [$LIMIT]

    Synology Client:
          --synology.user=      Synology username [$SYNOLOGY_USER]
          --synology.password=  Synology password [$SYNOLOGY_PASSWORD]
          --synology.url=       Synology URL (default: http://localhost:5000) [$SYNOLOGY_URL]
          --synology.insecure   Disable TLS (HTTPS) verification [$SYNOLOGY_INSECURE]
          --synology.timeout=   Default timeout (default: 30s) [$SYNOLOGY_TIMEOUT]
```
