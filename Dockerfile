FROM golang:alpine

RUN apk --no-cache add git

ENV GOPATH=/opt

COPY . $GOPATH/src/github.com/Tweddle-SE-Team/goaws
WORKDIR $GOPATH/src/github.com/Tweddle-SE-Team/goaws

RUN go get -u github.com/aws/aws-sdk-go/... && \
    go get github.com/stretchr/testify && \
    go get ./... && \
    go test ./... && \
    go build -o /usr/bin/goaws .

COPY config/config.yaml /etc/goaws/

EXPOSE 4100

ENTRYPOINT ["/usr/bin/goaws"]
