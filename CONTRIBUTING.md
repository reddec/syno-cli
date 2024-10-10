## Dev environment

Requires:

- Synology NAS with DSM 7+
- Go 1.23+
- Make

Place in `.env` configuration

```env
SYNOLOGY_URL=
SYNOLOGY_USER=
SYNOLOGY_PASSWORD=
```

it `.gitignore`'d and automatically loaded in tests.


### Generate test certificates

    make gen-certs