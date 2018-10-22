#!/bin/bash
set -euf -o pipefail
(
~/Arduino/nbiot-e2e/scripts/check_for_updates.sh ~/Arduino/libraries/ArduinoNBIoT | ts
echo restart arduino service | ts
sudo systemctl stop arduino | ts
sudo systemctl start arduino | ts
) &>> ~/log/arduino-nbiot-lib.log
