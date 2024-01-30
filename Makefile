.PHONY: all build clean protoc proto-gen

# Specify the paths
PROTO_DIR = ./grpc
GENERATED_DIR = ./gen
GO_OUT_DIR = ./pkg

# Specify the proto files
PROTO_FILES = $(wildcard $(PROTO_DIR)/*.proto)

clean:
	rm -rf $(GENERATED_DIR)

# Generate the gRPC code using protoc
protoc:
	mkdir -p $(GENERATED_DIR)
	protoc --go_out=$(GENERATED_DIR) --go_opt=paths=source_relative --go-grpc_out=$(GENERATED_DIR) --go-grpc_opt=paths=source_relative cryptobridge.proto

# Example: Run the server
run: 
	go run main.go

set-path:
	export PATH=$PATH:$(go env GOPATH)/bin