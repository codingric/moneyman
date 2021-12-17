##
# checkov:skip=CKV_DOCKER_2:Build stage image
# checkov:skip=CKV_DOCKER_3:Build stage image

FROM golang:alpine AS build

WORKDIR /app

RUN apk add gcc libc-dev git

COPY . .

RUN go get \
    && go build -o mailparser

FROM alpine:3.14
LABEL org.opencontainers.image.source https://github.com/codingric/moneyman

ENTRYPOINT [ "/app/mailparser" ]
CMD ["-v"]

COPY --from=build /app/mailparser .

