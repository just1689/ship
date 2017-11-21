FROM golang:1.9.1-alpine

RUN mkdir -p /go/src/github.com/SprintHive/ship
WORKDIR /go/src/github.com/SprintHive/ship

RUN apk update
RUN apk add curl git gcc libc-dev
RUN curl -OL https://github.com/golang/dep/releases/download/v0.3.2/dep-linux-amd64
RUN mv dep-linux-amd64 /usr/bin/dep
RUN chmod a+x /usr/bin/dep
