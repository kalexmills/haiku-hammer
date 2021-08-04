CGO_ENABLED=1 GOARCH=amd64 GOOS=linux go build -o bin/haiku-hammer src/server/main.go
cp bin/haiku-hammer deploy/server