## Dev environment

Requires:

- Synology NAS with DSM 7+
- Go 1.23+
- Make
- [Linter](https://golangci-lint.run/) 1.61.0 +

Place in `.env` configuration

```env
SYNOLOGY_URL=
SYNOLOGY_USER=
SYNOLOGY_PASSWORD=
```

you may use [direnv](https://direnv.net/) for automatic load

it `.gitignore`'d and automatically loaded in tests.


### Generate test certificates

    make gen-certs


### Generate code

    make generate

### Run linters

    make lint