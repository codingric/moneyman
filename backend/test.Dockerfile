FROM golang:alpine

RUN apk add gcc libc-dev git

WORKDIR /build

COPY . .
RUN go mod download

CMD ["go", "test", "-v", "-cover"]