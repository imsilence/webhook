FROM --platform=amd64 golang:latest as builder

LABEL maintainer imsilence@outlook.com

WORKDIR /opt/webhook/


COPY . .

RUN go env -w GO111MODULE=on && \
    go env -w GOPROXY=https://goproxy.cn,direct && \
    go build .

FROM --platform=amd64 centos:latest

LABEL maintainer imssilence@outlook.com

WORKDIR /opt/webhook

COPY --from=builder /opt/webhook/webhook .

ENTRYPOINT ["/opt/webhook/webhook"]