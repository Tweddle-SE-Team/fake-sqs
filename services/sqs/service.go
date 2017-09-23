package sqs

import (
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"github.com/Tweddle-SE-Team/goaws/services/common"
	"github.com/Tweddle-SE-Team/goaws/services/common/queue"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Message struct

type Message struct {
	MessageBody            []byte                `xml:"Body,omitempty"`
	MessageAttributes      []SqsMessageAttribute `xml:"MessageAttribute,omitempty"`
	MessageId              string                `xml:"MessageId,omitempty"`
	MD5OfMessageAttributes string                `xml:"MD5OfMessageAttributes,omitempty"`
	MD5OfMessageBody       string                `xml:"MD5OfBody,omitempty"`
	ReceiptHandle          string                `xml:"ReceiptHandle,omitempty"`
	ReceiptTime            time.Time
}

func GetQueueNameFromRequest(request *http.Request) string {
	if request.FormValue("QueueUrl") != "" {
		uriSegments := strings.Split(request.FormValue("QueueUrl"), "/")
		return uriSegments[len(uriSegments)-1]
	}
	u, err := url.Parse(request.URL.String())
	if err != nil {
		vars := mux.Vars(request)
		return vars["queueName"]
	}
	uriSegments := strings.Split(u.Path, "/")
	return uriSegments[len(uriSegments)-1]
}

func NewMessage(MessageBody []byte, MessageAttributes []SqsMessageAttribute, ReceiptHandle string, md5OfMessageAttributes string) *Message {
	MessageId, _ := common.NewUUID()
	return &Message{
		MessageBody:            MessageBody,
		MessageAttributes:      MessageAttributes,
		MessageId:              MessageId,
		ReceiptTime:            time.Now(),
		ReceiptHandle:          ReceiptHandle,
		MD5OfMessageBody:       common.GetMD5Hash(string(MessageBody[:])),
		MD5OfMessageAttributes: md5OfMessageAttributes}
}

func (c *Message) UpdateReceiptHandle() {
	c.ReceiptTime = time.Now()
	uuid, _ := common.NewUUID()
	c.ReceiptHandle = c.MessageId + uuid
}

// Queue struct

type Queue struct {
	Name        string
	URL         string
	Arn         string
	TimeoutSecs int
	Messages    *queue.BlockingQueue
}

func NewQueue(name string, host string) *Queue {
	return &Queue{
		Name:        name,
		URL:         fmt.Sprintf("http://%s/queue/%s", host, name),
		TimeoutSecs: 30,
		Arn:         fmt.Sprintf("http://%s/queue/%s", host, name),
		Messages:    queue.New()}
}

// SQS struct

type SQS struct {
	Queues *queue.BlockingQueue
}

func NewSQS() *SQS {
	return &SQS{Queues: queue.New()}
}

func (c *SQS) ExtractSqsMessageAttributes(request *http.Request) ([]SqsMessageAttribute, string) {
	attributes := make(map[string]SqsMessageAttribute)
	outputAttributes := make([]SqsMessageAttribute, 0, 0)
	for i := 1; true; i++ {
		name := request.FormValue(fmt.Sprintf("MessageAttribute.%d.Name", i))
		if name == "" {
			break
		}
		dataType := request.FormValue(fmt.Sprintf("MessageAttribute.%d.Value.DataType", i))
		if dataType == "" {
			log.Warnf("DataType of MessageAttribute %s is missing, MD5 checksum will most probably be wrong!\n", name)
			continue
		}
		for _, valueKey := range [...]string{"StringValue", "BinaryValue"} {
			value := request.FormValue(fmt.Sprintf("MessageAttribute.%d.Value.%s", i, valueKey))
			if value != "" {
				messageAttributeValue := SqsMessageAttributeValue{DataType: dataType}
				if valueKey == "StringValue" {
					messageAttributeValue.StringValue = value
				} else if valueKey == "BinaryValue" {
					messageAttributeValue.BinaryValue = value
				}
				attribute := SqsMessageAttribute{Name: name, Value: messageAttributeValue}
				attributes[name] = attribute
				outputAttributes = append(outputAttributes, attribute)
			}
		}
		if _, ok := attributes[name]; !ok {
			log.Warnf("StringValue or BinaryValue of MessageAttribute %s is missing, MD5 checksum will most probably be wrong!\n", name)
		}
	}
	return outputAttributes, c.HashAttributes(attributes)
}

func (c *SQS) HashAttributes(attributes map[string]SqsMessageAttribute) string {
	hasher := md5.New()
	keys := common.SortKeys(attributes)
	for _, key := range keys {
		attributeValue := attributes[key]
		common.AddStringToHash(hasher, key)
		common.AddStringToHash(hasher, attributeValue.Value.DataType)
		if attributeValue.Value.StringValue != "" {
			hasher.Write([]byte{1})
			common.AddStringToHash(hasher, attributeValue.Value.StringValue)
		} else if attributeValue.Value.BinaryValue != "" {
			hasher.Write([]byte{2})
			bytes, _ := base64.StdEncoding.DecodeString(attributeValue.Value.BinaryValue)
			common.AddBytesToHash(hasher, bytes)
		}
	}
	return hex.EncodeToString(hasher.Sum(nil))
}

var Service *SQS = NewSQS()
