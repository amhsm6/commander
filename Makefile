build:
	GOARCH=arm64 go build -o bin/commander -ldflags '-s -w' ./cmd/commander
	go build -o bin/comm -ldflags '-s -w' ./cmd/comm

dev:
	CI=1 CLICOLOR_FORCE=1 air

clean: clean-proto
	rm -rf bin


.PHONY: proto
proto:
	protoc -I=proto --go_out=pkg --go-grpc_out=pkg proto/*.proto

clean-proto:
	rm -rf pkg/pb

reproto: clean-proto proto
