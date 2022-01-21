FROM golang:alpine

WORKDIR /build

RUN apk add gcc libc-dev git

COPY . .
RUN go mod download

RUN find .

CMD ["go", "test", "-v", "-cover"]