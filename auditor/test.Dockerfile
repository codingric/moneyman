FROM golang:alpine

WORKDIR /build

RUN apk add gcc libc-dev git

COPY . .
RUN go mod download

CMD ["go", "test", "-gcflags=all=-l", "-v", "-cover"]