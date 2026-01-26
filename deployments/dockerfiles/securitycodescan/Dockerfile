FROM mcr.microsoft.com/dotnet/sdk:6.0-alpine

RUN apk add jq openssh bash

RUN dotnet tool install --global security-scan --version 5.6.7

# Install additional .NET SDK versions for compatibility (optional, skip errors)
RUN wget -q https://dot.net/v1/dotnet-install.sh && chmod +x dotnet-install.sh || true
RUN ./dotnet-install.sh -c 5.0 --install-dir /usr/share/dotnet || true
RUN ./dotnet-install.sh -c 7.0 --install-dir /usr/share/dotnet || true
RUN rm -f dotnet-install.sh || true

ENV PATH="$PATH:/root/.dotnet/tools"