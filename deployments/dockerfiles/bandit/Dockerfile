# Dockerfile used to create "husyci/bandit:3.6" image
# https://hub.docker.com/r/huskyci/bandit/

FROM python:3.13-alpine AS builder

# Install and compile bandit + its dependencies
RUN apk add --no-cache jq alpine-sdk
RUN pip install bandit

# Now build a fresh container, copying across the compiled pieces
FROM python:3.13-alpine

COPY --from=builder /usr/local/lib/python3.13 /usr/local/lib/python3.13
COPY --from=builder /usr/local/bin/bandit /usr/local/bin/bandit
COPY husky-file-ignore.sh /usr/local/bin/husky-file-ignore.sh

# Add packages that we need in the final image on runtime
RUN apk add --no-cache bash git jq openssh-client 
