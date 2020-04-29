OUTPUT_DIR=./bin
VERSION=`cat VERSION`
ENDPOINT=http://localhost:8080
#ENDPOINT=https://api-v2.capturoo.com
GIT_COMMIT=`git rev-list -1 HEAD | cut -c1-8`

build:
	@go build -o bin/capturoo -ldflags "-X 'main.version=${VERSION}' -X 'main.endpoint=${ENDPOINT}' -X 'main.firebaseAPIKey=${FIREBASE_API_KEY}' -X 'main.gitCommit=${GIT_COMMIT}'" cmd/capturoo/main.go

mac-cli:
	@GOOS=darwin GOARCH=amd64 go build -o bin/capturoo-darwin-amd64 -ldflags "-X 'main.version=${VERSION}' -X 'main.endpoint=${ENDPOINT}' -X 'main.firebaseAPIKey=${FIREBASE_API_KEY}' -X 'main.gitCommit=${GIT_COMMIT}'" cmd/capturoo/main.go

clean:
	-@rm -r $(OUTPUT_DIR)/* 2> /dev/null || true
