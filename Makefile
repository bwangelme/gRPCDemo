.PHONY: protoc build

protoc:
	@protoc --go_out=. \
	  --go_opt=paths=source_relative \
	  --go-grpc_out=. \
	  --go-grpc_opt=paths=source_relative \
	  pb/routeguide.proto

build:
	@go build ./cmd/cli/
	@go build ./cmd/svc/
