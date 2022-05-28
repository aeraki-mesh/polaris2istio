
.PHONY: build tools test mod-update

build:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o build/bin/main cmd/polaris2istio/main.go

tools:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o build/bin/polaris-server-mock cmd/tools/polaris-server-mock/polaris-server-mock.go

test:
	go test -v ./...

mod-update:
	go get -u ./...
	go mod tidy