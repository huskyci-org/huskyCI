# Dockerfile used to create "husyci/tfsec" image
# https://hub.docker.com/r/huskyci/tfsec/
FROM ghcr.io/aquasecurity/tfsec:v1.28.14

USER root

RUN apk update && apk upgrade \
	&& apk add git jq openssh-client curl