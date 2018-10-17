#!/bin/bash
set -e

usage() {
    echo -n "usage: ${0} arduino_sketch_dir arduino_board arduino_port [arduino_board arduino_port [...]]

Verify and upload arduino sketch to a board"
}

if (( $# < 3 || !($# % 2) ))
then
    echo Wrong number of arguments: $#
    usage
    exit 1
fi

DIR=$1
shift

# for each arduino board and port
while [[ $# -gt 1 ]]; do
    ARDUINO_BOARD=$1
    ARDUINO_PORT=$2
    shift 2
    arduino-cli compile --fqbn $ARDUINO_BOARD $DIR
    arduino-cli upload -p $ARDUINO_PORT --fqbn $ARDUINO_BOARD $DIR
done
