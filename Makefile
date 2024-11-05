.PHONY: driver
driver: 
	mkdir -p internal/driver/driver/bin
	go build -o internal/driver/driver/bin/driver internal/driver/main.go

build: driver
	go build

install: driver
	go install
