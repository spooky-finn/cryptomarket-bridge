# be free to contribute
1. export PATH="$PATH:$(go env GOPATH)/bin"
2. rotoc --go_out=./cryptobridge --go_opt=paths=source_relative --go-grpc_out=./cryptobridge --go-grpc_opt=paths=source_relative cryptobridge.proto
