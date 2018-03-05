set -e


# build piftester
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ./piftester ./piftester.go
docker build -t mfenwickbd/piftester:latest .
docker push mfenwickbd/piftester:latest
