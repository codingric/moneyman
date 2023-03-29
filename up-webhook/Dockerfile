FROM golang:alpine AS build

WORKDIR /build

RUN apk add gcc libc-dev git

COPY go.sum go.mod ./
RUN go mod download -x

COPY . .
RUN go build -o up-webhook

FROM alpine:3.16
LABEL org.opencontainers.image.source https://github.com/codingric/moneyman

RUN apk add tzdata \
    && echo "Australia/Melbourne" >  /etc/timezone

ENTRYPOINT [ "/usr/local/bin/up-webhook" ]

COPY --from=build /build/up-webhook /usr/local/bin/

