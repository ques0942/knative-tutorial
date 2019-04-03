FROM golang:1.11.6-alpine

WORKDIR /go/src/app
ADD main.go /go/src/app/main.go
ADD Gopkg.toml /go/src/app/Gopkg.toml

RUN apk update && \
    apk add --no-cache git && \
    go get -u github.com/golang/dep/cmd/dep && \
    dep ensure

CMD ["go", "run", "main.go"]