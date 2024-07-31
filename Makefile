all: clean build run

up:
	goose -dir migrations sqlite3 "file:dmv.db?_loc=auto" up

down:
	goose -dir migrations sqlite3 "file:dmv.db?_loc=auto" down

new:
	@ if [ -z $(name) ]; then echo "name is required"; exit 1; fi
	goose -dir migrations create $(name) sql

build:
	go build -o dmv .

clean:
	rm -f dmv dmv.db

run: build
	./dmv

help:
	@echo "make build - build the binary"
	@echo "make clean - remove the binary"
	@echo "make run - build and run the binary"
	@echo "make all - clean, build and run the binary"
	@echo "make up - run the migrations"
	@echo "make down - rollback the migrations"
	@echo "make new name=NAME - create a new migration"
	@echo "make help - display this help message"
