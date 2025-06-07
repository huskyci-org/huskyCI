# Dockerfile used to create "huskyci/npmaudit" image
# https://hub.docker.com/r/huskyci/npmaudit/

FROM node:24-alpine

RUN apk update && apk upgrade \
	&& apk add --no-cache alpine-sdk bash git openssh-client \
	&& rm -rf /var/cache/apk/* \
	&& npm install -g npm@11.3.0
ADD https://github.com/stedolan/jq/releases/download/jq-1.5/jq-linux64 /jq
RUN chmod +x ./jq && cp jq /usr/bin
