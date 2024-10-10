lint:
	golangci-lint run

snapshot:
	goreleaser --rm-dist --snapshot

local:
	rm -rf dist
	mkdir -p dist
	CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o dist/syno-cli ./cmd/syno-cli
	cp Dockerfile.release dist/Dockerfile
	cd dist && docker build -t syno-cli .

# Generate test certs
# https://github.com/FiloSottile/mkcert
gen-certs: test-data
	mkdir -p test-data
	cd test-data && CAROOT=. go run filippo.io/mkcert@v1.4.4 example.com


# Generate go code
generate:
	go generate ./...

test:
	go test -v ./...

.PHONY: generate test