#!/bin/bash

# Generate C and Go protocol buffer code.

# https://stackoverflow.com/questions/59895/getting-the-source-directory-of-a-bash-script-from-within
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null && pwd )"
cd $DIR

# build local protoc-gen-go from the protobuf version we depend on
(cd `go list -f '{{.Dir}}' -m github.com/golang/protobuf`/protoc-gen-go && go build -o $DIR/protoc-gen-go)

OPTS="\
	--plugin=protoc-gen-nanopb=nanopb/generator/protoc-gen-nanopb \
	--plugin=protoc-gen-go=protoc-gen-go \
	--nanopb_out=../device \
	--go_out=paths=source_relative,Mnanopb/generator/proto/nanopb.proto=github.com/telenordigital/nbiot-e2e/server/pb/nanopb/generator/proto:../server/pb"

protoc $OPTS \
	nanopb/generator/proto/nanopb.proto

protoc $OPTS \
	message.proto
