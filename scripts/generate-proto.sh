#!/bin/bash

# Generate Go code from protobuf files

PROTO_DIR="proto"
OUTPUT_DIR="proto"

# Install protoc-gen-go if not present
if ! command -v protoc-gen-go &> /dev/null; then
    go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
fi

# Generate budget service
protoc --go_out=$OUTPUT_DIR --go_opt=paths=source_relative \
       --go-grpc_out=$OUTPUT_DIR --go-grpc_opt=paths=source_relative \
       $PROTO_DIR/budget/budget.proto

# Generate events
protoc --go_out=$OUTPUT_DIR --go_opt=paths=source_relative \
       $PROTO_DIR/events/events.proto

echo "Protobuf code generation complete!"

