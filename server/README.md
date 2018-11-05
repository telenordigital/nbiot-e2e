# e2e server

This server monitors a device collection and sends an alert when it detects that device failed to send a mesage or when it receives a duplicate message.

Set the `TELENOR_NBIOT_TOKEN` environment variable to your API token before running the server.

To configure the service the first time, run this on the server:

    sudo cp e2e.service /etc/systemd/system/
    sudo systemctl enable e2e # start service after boot

# Deploy a new version

    make deploy

# Watch log (remotely)

    ssh ubuntu@e2e.nbiot.engineering journalctl -f -u e2e
