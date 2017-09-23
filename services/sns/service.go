package sns

import (
	"github.com/Tweddle-SE-Team/goaws/services/common"
	"github.com/Tweddle-SE-Team/goaws/services/common/queue"
	"strings"
)

type (
	Protocol         string
	MessageStructure string
)

const (
	ProtocolSQS          Protocol         = "sqs"
	ProtocolDefault      Protocol         = "default"
	MessageStructureJson MessageStructure = "json"
)

// Subscription struct

type Subscription struct {
	TopicArn        string
	Protocol        string
	SubscriptionArn string
	EndPoint        string
	Raw             bool
}

func NewSubscription(topicArn string, protocol string, endpoint string, raw bool) *Subscription {
	subscriptionUuid, _ := common.NewUUID()
	subscriptionArn := topicArn + ":" + subscriptionUuid
	return &Subscription{
		TopicArn:        topicArn,
		Protocol:        protocol,
		SubscriptionArn: subscriptionArn,
		EndPoint:        endpoint,
		Raw:             raw}
}

func (c *Subscription) GetTopicName() string {
	uriSegments := strings.Split(c.TopicArn, ":")
	return uriSegments[len(uriSegments)-1]
}

func (c *Subscription) getQueueName() string {
	uriSegments := strings.Split(c.EndPoint, "/")
	return uriSegments[len(uriSegments)-1]
}

// Topic struct

type Topic struct {
	Name          string
	Arn           string
	Subscriptions *queue.BlockingQueue
}

func NewTopic(arn *string, name *string) *Topic {
	topicName := ""
	topicArn := ""
	if arn == nil && name != nil {
		topicName = *name
		topicArn = "arn:aws:sns:local:000000000000:" + topicName
	} else if arn != nil && name == nil {
		topicArn = *arn
		uriSegments := strings.Split(topicArn, ":")
		topicName = uriSegments[len(uriSegments)-1]
	}
	return &Topic{Arn: topicArn, Name: topicName, Subscriptions: queue.New()}
}

// SNS struct

type SNS struct {
	Topics *queue.BlockingQueue
}

func NewSNS() *SNS {
	return &SNS{Topics: queue.New()}
}

var Service *SNS = NewSNS()
