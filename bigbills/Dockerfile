##
# checkov:skip=CKV_DOCKER_2:Build stage image
# checkov:skip=CKV_DOCKER_3:Build stage image

FROM golang:alpine AS build

WORKDIR /app

RUN apk add gcc libc-dev

COPY go.sum .
COPY go.mod .
RUN go mod download

COPY *.go ./
RUN go build

FROM alpine:3.14
LABEL org.opencontainers.image.source https://gitlab.com/codingric/moneyman

RUN apk add tzdata \
    && echo "Australia/Melbourne" >  /etc/timezone

ENTRYPOINT [ "/bigbills" ]
CMD ["check"]

COPY --from=build /app/bigbills .

