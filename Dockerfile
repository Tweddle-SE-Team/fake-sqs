FROM golang:alpine

RUN apk --no-cache add git

RUN go get github.com/Masterminds/glide && \
    go install github.com/Masterminds/glide

ENV GOPATH=/opt

COPY glide.yaml $GOPATH/src/github.com/Tweddle-SE-Team/goaws/

WORKDIR $GOPATH/src/github.com/Tweddle-SE-Team/goaws

RUN glide install

COPY . $GOPATH/src/github.com/Tweddle-SE-Team/goaws

RUN go test ./... && \
    go build -o /usr/bin/goaws github.com/Tweddle-SE-Team/goaws

COPY config/config.yaml /etc/goaws/

EXPOSE 4100

ENTRYPOINT ["/usr/bin/goaws"]
