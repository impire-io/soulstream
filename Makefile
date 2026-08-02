.PHONY: fmt tidy build test lint check

# Format all Go source (gofmt); golangci-lint's formatters also cover goimports.
# gofmt is file-based, so the root invocation covers the node/ module too.
fmt:
	gofmt -w .

# Keep go.mod/go.sum honest — both modules.
tidy:
	go mod tidy
	cd node && go mod tidy

build:
	go build ./...
	go build -o bin/ ./cmd/...
	cd node && go build ./... && go build -o ../bin/ ./cmd/...

# All tests, no skips — both modules.
test:
	go test ./...
	cd node && go test ./...

lint:
	golangci-lint run
	cd node && golangci-lint run

# The one gate to run before every commit: everything green.
check: fmt tidy build test lint
