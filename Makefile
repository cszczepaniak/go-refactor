.PHONY: driver
driver: 
	go build -o bin/driver internal/driver/main.go

build: driver
	go build
