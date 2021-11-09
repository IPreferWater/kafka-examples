ARG GO_VERSION=1.14

FROM golang:${GO_VERSION}-alpine AS builder

RUN apk update && apk add alpine-sdk git && rm -rf /var/cache/apk/*

RUN mkdir -p /go-producer
WORKDIR /go-producer

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .
RUN go build -o ./app -tags musl

EXPOSE 3000

ENTRYPOINT ["./app"]