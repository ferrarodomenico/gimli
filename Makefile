build:
	@go build -o bin/gimli cmd/main/main.go

run: build
	@./bin/gimli

rungc: build
	GODEBUG=gctrace=1 ./bin/gimli 2> gc_trace.log

lt:
	@go run cmd/loadtest/main.go

.PHONY: build run lt