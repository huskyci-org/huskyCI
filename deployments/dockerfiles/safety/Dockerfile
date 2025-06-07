# filepath: /home/guilhermedesouzaferreira/Gits/huskyCI/deployments/dockerfiles/safety/Dockerfile
FROM python:3.13-alpine

# Instalar dependências necessárias para compilar psutil
RUN apk add --no-cache \
    bash \
    gcc \
    jq \
    libffi-dev \
    linux-headers \
    musl-dev \
    openssh-client \
    openssl-dev \
    python3-dev && \
    pip install --no-cache-dir safety