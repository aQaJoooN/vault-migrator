.PHONY: build clean test install

build:
	go build -o vault-migrator .

install:
	go install .

clean:
	rm -f vault-migrator vault-migrator.exe

test:
	go test ./...

deps:
	go mod download
	go mod tidy
