.PHONY: driver
driver: 
	mkdir -p internal/driver/setup/bin
	go build -o internal/driver/setup/bin/driver internal/driver/main.go

build: driver
	go build
