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

echo Starting upload of new code

# for each arduino board and port
while [[ $# -gt 1 ]]; do
    ARDUINO_BOARD=$1
    ARDUINO_PORT=$2
    shift 2

    # kill process using the serial port
    fuser -k $ARDUINO_PORT || true

    arduino-cli compile --fqbn $ARDUINO_BOARD $DIR
    arduino-cli upload -p $ARDUINO_PORT --fqbn $ARDUINO_BOARD $DIR

    # log serial port output
    socat $ARDUINO_PORT,b9600,raw - &>> ~/log/`basename $ARDUINO_PORT`.log &
done

echo Done uploading
