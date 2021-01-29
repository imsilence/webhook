FROM --platform=amd64 golang:latest as builder

LABEL maintainer imsilence@outlook.com

ARG VERSION=v0.0.0

WORKDIR /opt/cmdb/


COPY . .

RUN go build -ldflags "-X main.Version=${VERSION}" .

FROM --platform=amd64 centos:latest

LABEL maintainer imssilence@outlook.com

WORKDIR /opt/cmdb

COPY --from=builder /opt/cmdb/cmdb .

ENTRYPOINT ["/opt/cmdb/cmdb"]
