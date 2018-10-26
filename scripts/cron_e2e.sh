#!/bin/bash
set -euf -o pipefail
(
~/Arduino/nbiot-e2e/scripts/check_for_updates.sh ~/Arduino/nbiot-e2e | ts
cd ~/Arduino/nbiot-e2e/arduino-service
go build | ts
echo restart arduino service | ts
sudo systemctl stop arduino | ts
sudo systemctl start arduino | ts
) &>> ~/log/nbiot-e2e.log
