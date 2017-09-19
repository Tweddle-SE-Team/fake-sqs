FROM golang:alpine

RUN apk --no-cache add git
RUN go get github.com/Tweddle-SE-Team/goaws/...

COPY app /opt/app
WORKDIR /opt/app
RUN go build -o goaws /opt/app/cmd/goaws.go

COPY app/conf/goaws.yaml /conf/

EXPOSE 4100

ENTRYPOINT ["/opt/app/goaws"]
