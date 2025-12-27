.PHONY: build install test clean

build:
	go build -o bin/synheart cmd/synheart/main.go

install:
	go install ./cmd/synheart

test:
	go test ./...

clean:
	rm -rf bin/
