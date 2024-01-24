
build:
	go mod download
	go build .\cmd\shortener\main.go
	go build .\cmd\staticlint\mycheck.go
test:
	go test ./... -coverprofile cover.out
	go tool cover -func cover.out
linter: build
	./mycheck.exe -shadow=false ./pkg/server/server.go
	./mycheck.exe -all ./cmd/shortener/main.go
	./mycheck.exe -all ./pkg/storage/...
run: test build
	./main.exe
all: test linter build run