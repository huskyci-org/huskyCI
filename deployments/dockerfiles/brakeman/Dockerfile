# Dockerfile used to create "husyci/brakeman" image
# https://hub.docker.com/r/huskyci/brakeman/

FROM ruby:3.4-alpine

RUN apk update && apk upgrade \
	&& apk add --no-cache alpine-sdk bash git openssh-client \
	&& gem install brakeman \
	&& wget -O jq https://github.com/jqlang/jq/releases/download/jq-1.7.1/jq-linux-amd64 \
	&& chmod +x ./jq \
	&& cp jq /usr/bin \
	&& rm jq
