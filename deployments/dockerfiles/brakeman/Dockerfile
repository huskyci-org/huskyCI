# Dockerfile used to create "husyci/brakeman" image
# https://hub.docker.com/r/huskyci/brakeman/

FROM ruby:3.4-alpine

RUN apk update && apk upgrade \
	&& apk add --no-cache alpine-sdk bash openssh-client \
	&& apk add git wget

RUN gem install brakeman -v 7.1.2

RUN wget -O jq https://github.com/stedolan/jq/releases/download/jq-1.7/jq-linux64
RUN chmod +x ./jq
RUN cp jq /usr/bin
