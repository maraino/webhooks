all: build db

build:
	go build -o bin/webhooks main.go

run:
	go run main.go

db:
	dbmate --url "sqlite:db/database.sqlite3" --no-dump-schema up

.PHONY: all build run db