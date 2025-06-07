# Dockerfile used to create "husyci/enry" image
# https://hub.docker.com/r/huskyci/enry/
FROM golang:alpine AS builder

RUN apk update && apk upgrade \
	&& apk add --update --no-cache git openssh-client

RUN git clone https://github.com/go-enry/enry.git && \
    cd enry && \
	go mod tidy && \
	CGO_ENABLED=0 go build -o enry && \
	cp enry /usr/bin && \
	rm -rf enry

FROM alpine:3.21

RUN apk update && apk upgrade \
	&& apk add --update --no-cache curl git openssh-client && \
	rm -rf /var/cache/apk/*

COPY --from=builder /usr/bin/enry /usr/bin
