##
# checkov:skip=CKV_DOCKER_2:Build stage image
# checkov:skip=CKV_DOCKER_3:Build stage image

FROM golang:alpine AS build

WORKDIR /app

RUN apk add gcc libc-dev

COPY *.go ./
COPY go.sum .
COPY go.mod .

RUN go mod download \
    && go build

FROM alpine:3.14
LABEL org.opencontainers.image.source https://gitlab.com/codingric/moneyman/go/bigbills

ENTRYPOINT [ "/bigbills" ]
CMD ["check"]

COPY --from=build /app/bigbills .

