FROM golang:alpine AS build

WORKDIR /build

RUN apk add gcc libc-dev git

RUN go install github.com/mattn/go-sqlite3

COPY go.* .
RUN go mod download

COPY . .

RUN go build -o backend

FROM alpine:3.14
LABEL org.opencontainers.image.source https://github.com/codingric/moneyman

RUN apk add tzdata \
    && echo "Australia/Melbourne" >  /etc/timezone

ENTRYPOINT [ "/backend" ]
CMD ["-v"]

COPY --from=build /build/backend .

