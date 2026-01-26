# Dockerfile used to create "huskyci/npmaudit" image
# https://hub.docker.com/r/huskyci/npmaudit/

FROM node:lts-alpine

RUN apk update && apk upgrade \
	&& apk add --no-cache alpine-sdk bash openssh-client \
	&& apk add git wget

RUN npm install -g npm@11.8.0
RUN wget -O jq https://github.com/stedolan/jq/releases/download/jq-1.7/jq-linux64
RUN chmod +x ./jq
RUN cp jq /usr/bin
