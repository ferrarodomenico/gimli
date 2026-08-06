build:
	@go build -o bin/gimli cmd/main/main.go

run: build
	@./bin/gimli

lt:
	@go run cmd/loadtest/main.go

.PHONY: build run lt