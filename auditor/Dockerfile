##
# checkov:skip=CKV_DOCKER_2:Build stage image
# checkov:skip=CKV_DOCKER_3:Build stage image

FROM golang:alpine AS build

WORKDIR /build

RUN apk add gcc libc-dev git

COPY go.sum go.mod ./
RUN go mod download

COPY . .
RUN go build -o auditor

FROM alpine:3.14
LABEL org.opencontainers.image.source https://gitlab.com/codingric/moneyman

RUN apk add tzdata \
    && echo "Australia/Melbourne" >  /etc/timezone

ENTRYPOINT [ "/usr/bin/auditor" ]
CMD ["-v"]

COPY --from=build /build/auditor /usr/bin/

