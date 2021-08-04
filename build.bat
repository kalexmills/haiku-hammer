SET GOARCH=386
SET GOOS=linux
SET CGO_ENABLED=1
go build -o bin/haiku-hammer src/server/main.go
xcopy /y .\bin\haiku-hammer .\deploy\server