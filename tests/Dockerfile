FROM golang:1.7-alpine

RUN apk add --update \
		git \
		&& rm -rf /var/cache/apk/*
RUN mkdir -p /go/src/github.com/docker/docker-e2e
WORKDIR /go/src/github.com/docker/docker-e2e

COPY . /go/src/github.com/docker/docker-e2e
RUN go get -v -d -t ./...

CMD ["go", "test", "-v"]
