# Dockerfile used to create "huskyciorg/trivy" image
# https://hub.docker.com/r/aquasec/trivy
FROM aquasec/trivy:0.68.2

USER root

RUN apk update && apk upgrade \
    && apk add curl git jq openssh-client \
    && rm -rf /var/cache/apk/* 