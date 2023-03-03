FROM golang:1.19 as builder

LABEL maintainer="NJWS, Inc."

WORKDIR /src/

COPY . .

RUN go mod download

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /build/proxy ./cmd/proxy

FROM ubuntu:18.04

LABEL maintainer="NJWS, Inc."

RUN apt update && \
    apt install ca-certificates curl -y && \
    rm -rf /var/lib/apt/lists/*

COPY --from=builder /build/proxy /usr/bin/

RUN chmod +x /usr/bin/proxy

CMD ["/usr/bin/proxy", "--debug"]
