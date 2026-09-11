FROM golang:1.25-alpine as buildbase

RUN apk add git build-base

WORKDIR /go/src/github.com/Bridgeless-Project/tss-wrapper-svc

ENV GO111MODULE="on"
ENV CGO_ENABLED=1
ENV GOOS="linux"
ENV GOPRIVATE=github.com/*
ENV GONOSUMDB=github.com/*
ENV GONOPROXY=github.com/*

COPY ./go.mod ./go.sum ./
RUN go mod download

COPY . .

RUN go build  -o /usr/local/bin/tss-wrapper-svc /go/src/github.com/Bridgeless-Project/tss-wrapper-svc


FROM alpine:3.9

COPY --from=buildbase /usr/local/bin/tss-wrapper-svc /usr/local/bin/tss-wrapper-svc

RUN apk add --no-cache ca-certificates

ENTRYPOINT ["tss-wrapper-svc"]