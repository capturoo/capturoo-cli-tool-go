OUTPUT_DIR=./bin
VERSION=`cat VERSION`
#ENDPOINT=https://api-staging.capturoo.com
ENDPOINT=http://localhost:8080
GIT_COMMIT=`git rev-list -1 HEAD | cut -c1-8`

build:
	@go build -o bin/capturoo -ldflags "-X 'main.version=${VERSION}' -X 'main.endpoint=${ENDPOINT}' -X 'main.gitCommit=${GIT_COMMIT}'" cmd/capturoo/main.go

mac-cli:
	@GOOS=darwin GOARCH=amd64 go build -o bin/capturoo-darwin-amd64 -ldflags "-X 'main.version=${VERSION}' -X 'main.endpoint=${ENDPOINT}' -X 'main.gitCommit=${GIT_COMMIT}'" cmd/capturoo/main.go

clean:
	-@rm -r $(OUTPUT_DIR)/* 2> /dev/null || true
