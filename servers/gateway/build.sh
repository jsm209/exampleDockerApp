GOOS=linux go build
docker build -t jsm209/gateway .
go clean
