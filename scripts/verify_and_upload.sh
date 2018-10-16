#!/bin/bash
set -e

usage() {
  echo -n "usage: ${0} arduino_sketch_dir arduino_board arduino_port

Verify and upload arduino sketch to a board"
}

if [[ "$#" -lt 3 ]]
then
  echo Too few arguments
  usage
  exit 1
fi

DIR=$1
ARDUINO_BOARD=$2
ARDUINO_PORT=$3

arduino-cli compile --fqbn $ARDUINO_BOARD ~/Arduino/nbiot-e2e/device
arduino-cli upload -p $ARDUINO_PORT --fqbn $ARDUINO_BOARD ~/Arduino/nbiot-e2e/device
