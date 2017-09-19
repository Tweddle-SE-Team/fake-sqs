FROM golang:alpine

RUN apk --no-cache add git

ENV GOPATH=/opt

COPY . $GOPATH/src/github.com/Tweddle-SE-Team/goaws
WORKDIR $GOPATH/src/github.com/Tweddle-SE-Team/goaws

RUN go get ./... && go build -o /usr/bin/goaws .

COPY config/config.yaml /etc/goaws/

EXPOSE 4100

ENTRYPOINT ["/usr/bin/goaws"]
