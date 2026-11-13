.PHONY: build run test vet tidy clean

# build compiles the static binary (CGO off — see docs/adr/0002).
build:
	cd backend && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o ../bin/openjob ./cmd/openjob

# run starts the server locally (without Docker) on :8080.
run: build
	OPENJOB_PORT=8080 ./bin/openjob

test:
	cd backend && go test ./...

vet:
	cd backend && go vet ./...

tidy:
	cd backend && go mod tidy

clean:
	rm -f bin/openjob
