.PHONY: build install test dev clean

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
