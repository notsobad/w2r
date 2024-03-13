.PHONY: all windows linux mac

BINARY_NAME=w2r
VERSION=1.0.0
LDFLAGS=-ldflags "-X main.Version=${VERSION}"

all: windows linux mac

windows:
	GOOS=windows GOARCH=amd64 go build ${LDFLAGS} -o ${BINARY_NAME}-windows-amd64.exe
	GOOS=windows GOARCH=arm64 go build ${LDFLAGS} -o ${BINARY_NAME}-windows-arm64.exe

linux:
	GOOS=linux GOARCH=amd64 go build ${LDFLAGS} -o ${BINARY_NAME}-linux-amd64
	GOOS=linux GOARCH=arm64 go build ${LDFLAGS} -o ${BINARY_NAME}-linux-arm64

mac:
	GOOS=darwin GOARCH=amd64 go build ${LDFLAGS} -o ${BINARY_NAME}-darwin-amd64
	GOOS=darwin GOARCH=arm64 go build ${LDFLAGS} -o ${BINARY_NAME}-darwin-arm64