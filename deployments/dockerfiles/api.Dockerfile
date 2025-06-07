FROM golang:1.24-alpine AS builder

COPY api/ /go/src/github.com/huskyci-org/huskyCI/api/
WORKDIR /go/src/github.com/huskyci-org/huskyCI/api/

RUN go build -o huskyci-api-bin server.go

FROM alpine:3.21.3

WORKDIR /go/src/github.com/huskyci-org/huskyCI/api/
COPY --from=builder /go/src/github.com/huskyci-org/huskyCI/api/huskyci-api-bin .
COPY api/config.yaml .
COPY api/api-tls-cert.pem .
COPY api/api-tls-key.pem .

RUN chmod +x huskyci-api-bin