#!/bin/bash

# Generate C and Go protocol buffer code.

cd `go list -m -f={{.Dir}}`/pb

OPTS="\
	--plugin=protoc-gen-nanopb=nanopb/generator/protoc-gen-nanopb \
	--nanopb_out=../device \
	--go_out=paths=source_relative,Mnanopb/generator/proto/nanopb.proto=github.com/telenordigital/nbiot-e2e/server/pb/nanopb/generator/proto:../server/pb"

protoc $OPTS \
	nanopb/generator/proto/nanopb.proto

protoc $OPTS \
	message.proto
