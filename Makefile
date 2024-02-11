
build:
	go mod download
	go build .\cmd\shortener\main.go
	go build .\cmd\staticlint\mycheck.go

test:
	go test ./... -v -coverprofile cover.out
	go tool cover -func cover.out

linter: build
	./mycheck.exe -shadow=false ./pkg/server/server.go
	./mycheck.exe -all ./cmd/shortener/main.go
	./mycheck.exe -all ./pkg/storage/...

stylization: 
	goimports -local "github.com/Dorrrke/shortener-url" -w .\cmd\
	goimports -local "github.com/Dorrrke/shortener-url" -w .\internal\
	goimports -local "github.com/Dorrrke/shortener-url" -w .\pkg\
	
run: test build
	./main.exe

all: test linter build run