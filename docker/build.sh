#/bin/sh
CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -o ./.dist/app -a -ldflags '-extldflags "-static"' .
docker build --tag $IMAGE_TAG -f docker/dockerfile ./.dist