set -e

CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ./imagefacadewebserver imagefacadewebserver.go

docker build -t mfenwickbd/imagefacadewebserver .
docker push mfenwickbd/imagefacadewebserver:latest
