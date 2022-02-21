FROM golang:alpine AS build

WORKDIR /build

RUN apk add gcc libc-dev git

COPY go.sum go.mod ./
RUN go mod download

COPY . .
RUN go build -o bigbills

FROM alpine:3.14
LABEL org.opencontainers.image.source https://gitlab.com/codingric/moneyman

RUN apk add tzdata \
    && echo "Australia/Melbourne" >  /etc/timezone

ENTRYPOINT [ "/usr/bin/bigbills" ]
CMD ["check"]

COPY --from=build /build/bigbills /usr/bin/

