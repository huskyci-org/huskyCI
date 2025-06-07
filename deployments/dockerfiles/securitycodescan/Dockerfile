FROM mcr.microsoft.com/dotnet/sdk:6.0-alpine

RUN apk add --no-cache bash jq openssh && \
	dotnet tool install --global security-scan --version 5.6.2 && \
	wget https://dot.net/v1/dotnet-install.sh && \
	chmod +x dotnet-install.sh && \
	./dotnet-install.sh -c 3.1 --install-dir /usr/share/dotnet && \
	./dotnet-install.sh -c 5.0 --install-dir /usr/share/dotnet && \
	./dotnet-install.sh -c 7.0 --install-dir /usr/share/dotnet

ENV PATH="$PATH:/root/.dotnet/tools"