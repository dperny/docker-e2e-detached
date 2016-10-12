FROM golang:1.7-alpine

RUN apk add --update \
		git \
		&& rm -rf /var/cache/apk/*
RUN mkdir -p /go/src/github.com/docker/docker-e2e/testkit
WORKDIR /go/src/github.com/docker/docker-e2e/testkit

COPY . /go/src/github.com/docker/docker-e2e/testkit
RUN go get -v -d ./... && \
	go install -v

ENTRYPOINT ["testkit"]
