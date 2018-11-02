.PHONY: all
all:
	go build

.PHONY: cross-compile
cross-compile:
	go get -u github.com/karalabe/xgo
	xgo -go 1.11 --targets=linux/amd64 github.com/telenordigital/nbiot-e2e/server
