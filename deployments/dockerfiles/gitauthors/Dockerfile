# Dockerfile used to create "husyci/gitauthors" image
# https://hub.docker.com/r/huskyci/gitauthors/

FROM alpine:latest

RUN apk update && apk upgrade \
	&& apk add --update --no-cache git openssh-client
