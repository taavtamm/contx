.PHONY: build install test dev clean release

build:
	go build -o bin/contx .

install:
	go install .

test:
	go test ./...

dev:
	go run .

clean:
	rm -rf bin/

release:
	GOOS=darwin  GOARCH=arm64  go build -o bin/contx-darwin-arm64  .
	GOOS=darwin  GOARCH=amd64  go build -o bin/contx-darwin-amd64  .
	GOOS=linux   GOARCH=amd64  go build -o bin/contx-linux-amd64   .
	GOOS=linux   GOARCH=arm64  go build -o bin/contx-linux-arm64   .
