go-run:
  go run cmd/server/main.go

go-build:
  go build -o mcp_gitlab_review cmd/server/main.go

# Tidy Go modules
go-tidy:
    go mod tidy

# Update all Go dependencies
go-update:
    go get -u -t ./... && just go-tidy

build:
  just go-build

test:
  go test ./...

# Tidy Go modules (alias for go-tidy)
tidy:
    just go-tidy

