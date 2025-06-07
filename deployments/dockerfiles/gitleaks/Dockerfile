# Dockerfile used to create "husyci/gitleaks" image
# https://hub.docker.com/r/huskyci/gitleaks/

FROM zricethezav/gitleaks:v8.12.0

USER root

RUN apk --no-cache add ca-certificates jq openssh-client

ENTRYPOINT []
CMD ["/bin/sh"]