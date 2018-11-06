.PHONY: all
all:
	go build

.PHONY: cross-compile
cross-compile:
	GOOS=linux GOARCH=amd64 go build -o server-linux-amd64

.PHONY: deploy
deploy: cross-compile
	scp server-linux-amd64 ubuntu@e2e.nbiot.engineering:server-linux-amd64-new
	ssh ubuntu@e2e.nbiot.engineering "sudo systemctl stop e2e && mv server-linux-amd64-new server-linux-amd64 && sudo systemctl start e2e && journalctl -f -u e2e"
