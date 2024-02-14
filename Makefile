ifneq (,$(wildcard ./.env))
    include .env
    export
endif

DB_STRING="host=$(DB_HOST) user=$(DB_USER) password=$(DB_PASSWORD) dbname=$(DB_NAME) sslmode=disable"

build:
	go build -o bin/tarkov-build-optimiser-api ./cmd/tarkov-build-optimiser-api 

run: build
	./bin/tarkov-build-optimiser-api
@test $${name?Please set variable 'name'}

.require-name:
	@test $${name?Please set variable 'name'}

migrate_create: .require-name
	GOOSE_DRIVER=postgres GOOSE_DBSTRING=$(DB_STRING) goose -dir=./migrations create $(name) go

migrate_up:
	GOOSE_DRIVER=postgres GOOSE_DBSTRING=$(DB_STRING) goose -dir=./migrations up

migrate_down:
	GOOSE_DRIVER=postgres GOOSE_DBSTRING=$(DB_STRING) goose -dir=./migrations down

clean:
	go clean
	rm -f bin/tarkov-build-optimiser-api

