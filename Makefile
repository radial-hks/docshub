.PHONY: build server cli test clean

build: server cli

server:
	go build -o docshub-server ./cmd/server

cli:
	go build -o docshub ./cmd/cli

test:
	go test ./... -v

clean:
	rm -f docshub-server docshub
