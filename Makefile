BINARY := clustertui
VERSION ?= dev
LDFLAGS := -ldflags "-s -w -X github.com/christopherluey/clustertui/cmd.Version=$(VERSION)"

.PHONY: build run clean lint tidy

build:
	go build $(LDFLAGS) -o $(BINARY) .

run: build
	./$(BINARY)

clean:
	rm -f $(BINARY)

lint:
	golangci-lint run ./...

tidy:
	go mod tidy
