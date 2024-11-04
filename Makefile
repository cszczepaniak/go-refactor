.PHONY: driver
driver: 
	mkdir -p driver/bin/setup
	go build -o internal/driver/setup/bin/driver internal/driver/main.go

build: driver
	go build
