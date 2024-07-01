FROM golang:alpine AS build

WORKDIR /build

RUN apk add gcc libc-dev git

#RUN CGO_ENABLED=1 go install github.com/mattn/go-sqlite3@latest
COPY go.* .
RUN CGO_ENABLED=1 go mod download -x

COPY . .

RUN CGO_ENABLED=1 go build -o backend -x

FROM alpine:3
LABEL org.opencontainers.image.source https://github.com/codingric/moneyman

RUN apk add tzdata \
    && echo "Australia/Melbourne" >  /etc/timezone

ENTRYPOINT [ "/backend" ]
CMD ["-v"]

COPY --from=build /build/backend .

