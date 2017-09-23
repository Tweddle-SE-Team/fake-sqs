package sns

import (
	"encoding/json"
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	"net/http"
	"time"

	"github.com/Tweddle-SE-Team/goaws/services/common"
	"github.com/Tweddle-SE-Team/goaws/services/sqs"
)

func (c *SNS) CreateTopic(request *http.Request) (interface{}, string, error) {
	topicName := request.FormValue("Name")
	topicEquals := func(s interface{}, v interface{}) bool {
		src := s.(*Topic)
		value := v.(*string)
		return src.Name == *value
	}
	topic := c.Topics.Get(&topicName, topicEquals)
	if topic == nil {
		topic = NewTopic(nil, &topicName)
		c.Topics.Put(topic)
	}
	return NewCreateTopicResponse(CreateTopicResult{TopicArn: topic.(Topic).Arn}), "XML", nil
}

func (c *SNS) DeleteTopic(request *http.Request) (interface{}, string, error) {
	topicArn := request.FormValue("TopicArn")
	topicEquals := func(s interface{}, v interface{}) bool {
		src := s.(*Topic)
		value := v.(*string)
		return src.Name == *value
	}
	if c.Topics.Remove(&topicArn, topicEquals) {
		return NewDeleteTopicResponse(), "XML", nil
	} else {
		return nil, "XML", errors.New("TopicNotFound")
	}
}

func (c *SNS) ListSubscriptions(request *http.Request) (interface{}, string, error) {
	totalMemberResults := make([]TopicMemberResult, 0, 0)
	for topic := range c.Topics.Iterator() {
		for subscription := range topic.(*Topic).Subscriptions.Iterator() {
			topicMemberResult := TopicMemberResult{
				TopicArn:        topic.(*Topic).Arn,
				Protocol:        subscription.(Subscription).Protocol,
				SubscriptionArn: subscription.(Subscription).SubscriptionArn,
				Endpoint:        subscription.(Subscription).EndPoint}
			totalMemberResults = append(totalMemberResults, topicMemberResult)
		}
	}
	return NewListSubscriptionsResponse(
		ListSubscriptionsResult{
			Subscriptions: TopicSubscriptions{
				Member: totalMemberResults}}), "XML", nil
}

func (c *SNS) ListSubscriptionsByTopic(request *http.Request) (interface{}, string, error) {
	topicArn := request.FormValue("TopicArn")
	topicEquals := func(s interface{}, v interface{}) bool {
		src := s.(*Topic)
		value := v.(*string)
		return src.Arn == *value
	}
	topic := c.Topics.Get(&topicArn, topicEquals)
	if topic != nil {
		topicMemberResults := make([]TopicMemberResult, 0, 0)
		for subscription := range topic.(*Topic).Subscriptions.Iterator() {
			topicMemberResult := TopicMemberResult{
				TopicArn:        topic.(*Topic).Arn,
				Protocol:        subscription.(Subscription).Protocol,
				SubscriptionArn: subscription.(Subscription).SubscriptionArn,
				Endpoint:        subscription.(Subscription).EndPoint}
			topicMemberResults = append(topicMemberResults, topicMemberResult)
		}
		return NewListSubscriptionsByTopicResponse(
			ListSubscriptionsResult{
				Subscriptions: TopicSubscriptions{
					Member: topicMemberResults}}), "XML", nil
	} else {
		return nil, "XML", errors.New("TopicNotFound")
	}
}

func (c *SNS) ListTopics(request *http.Request) (interface{}, string, error) {
	topicArnResult := make([]TopicArnResult, 0, 0)
	for topic := range c.Topics.Iterator() {
		topicArn := TopicArnResult{TopicArn: topic.(*Topic).Arn}
		topicArnResult = append(topicArnResult, topicArn)
	}
	return NewListTopicsResponse(
		ListTopicsResult{
			Topics: TopicNamestype{
				Member: topicArnResult}}), "XML", nil
}

func (c *SNS) Publish(request *http.Request) (interface{}, string, error) {
	topicArn := request.FormValue("TopicArn")
	topicEquals := func(s interface{}, v interface{}) bool {
		src := s.(*Topic)
		value := v.(*string)
		return src.Arn == *value
	}
	topicMessage := NewTopicMessage(
		"Notification",
		request.FormValue("TopicArn"),
		request.FormValue("Message"),
		ExtractSnsMessageAttributes(request),
		request.FormValue("MessageStructure"),
		request.FormValue("Subject"))
	queueEquals := func(s interface{}, v interface{}) bool {
		src := s.(*sqs.Queue)
		value := v.(*string)
		return src.Name == *value
	}
	if topic := c.Topics.Get(&topicArn, topicEquals); topic != nil {
		for s := range topic.(*Topic).Subscriptions.Iterator() {
			subscription := s.(*Subscription)
			if Protocol(subscription.Protocol) == ProtocolSQS {
				messageString, err := topicMessage.toString()
				if err == nil {
					sqsMessage := sqs.NewMessage(messageString, make([]sqs.SqsMessageAttribute, 0, 0), "", "")
					sqsMessage.UpdateReceiptHandle()
					queueName := subscription.getQueueName()
					q := sqs.Service.Queues.Get(&queueName, queueEquals)
					if q != nil {
						queue := q.(*sqs.Queue)
						queue.Messages.Put(sqsMessage)
					}
				} else {
					return nil, "XML", err
				}
			}
		}
	} else {
		return nil, "XML", errors.New("TopicNotFound")
	}
	messageId, _ := common.NewUUID()
	return NewPublishResponse(PublishResult{MessageId: messageId}), "XML", nil
}

func (c *SNS) SetSubscriptionAttributes(request *http.Request) (interface{}, string, error) {
	subscriptionArn := request.FormValue("SubscriptionArn")
	attribute := request.FormValue("AttributeName")
	value := request.FormValue("AttributeValue")
	subscriptionEquals := func(s interface{}, v interface{}) bool {
		src := s.(*Subscription)
		value := v.(*string)
		return src.SubscriptionArn == *value
	}
	for t := range c.Topics.Iterator() {
		topic := t.(*Topic)
		s := topic.Subscriptions.Get(&subscriptionArn, subscriptionEquals)
		if s != nil {
			subscription := s.(*Subscription)
			if attribute == "RawMessageDelivery" {
				if value == "true" {
					subscription.Raw = true
				} else {
					subscription.Raw = false
				}
				return NewSetSubscriptionAttributesResponse(), "XML", nil
			}
		}
	}
	return nil, "XML", errors.New("SubscriptionNotFound")
}

func (c *SNS) Subscribe(request *http.Request) (interface{}, string, error) {
	subscription := NewSubscription(
		request.FormValue("TopicArn"),
		request.FormValue("Protocol"),
		request.FormValue("Endpoint"),
		false)
	topicEquals := func(s interface{}, v interface{}) bool {
		src := s.(*Topic)
		value := v.(*string)
		return src.Name == *value
	}
	topicName := subscription.GetTopicName()
	if t := c.Topics.Get(&topicName, topicEquals); t != nil {
		topic := t.(*Topic)
		topic.Subscriptions.Put(subscription)
		return NewSubscribeResponse(SubscribeResult{
			SubscriptionArn: subscription.SubscriptionArn}), "XML", nil
	} else {
		return nil, "XML", errors.New("TopicNotFound")
	}
}

func (c *SNS) Unsubscribe(request *http.Request) (interface{}, string, error) {
	subscriptionArn := request.FormValue("SubscriptionArn")
	subscriptionEquals := func(src interface{}, value interface{}) bool {
		return src.(Subscription).SubscriptionArn == value.(string)
	}
	for topic := range c.Topics.Iterator() {
		if topic.(*Topic).Subscriptions.Remove(&subscriptionArn, subscriptionEquals) {
			return NewUnsubscribeResponse(), "XML", nil
		}
	}
	return nil, "XML", errors.New("SubscriptionNotFound")
}

func ExtractSnsMessageAttributes(req *http.Request) []SnsMessageAttribute {
	attributes := make([]SnsMessageAttribute, 0, 0)

	for i := 1; true; i++ {
		name := req.FormValue(fmt.Sprintf("MessageAttributes.entry.%d.Name", i))
		if name == "" {
			break
		}
		dataType := req.FormValue(fmt.Sprintf("MessageAttributes.entry.%d.Value.DataType", i))
		if dataType == "" {
			log.Warnf("DataType of MessageAttribute %s is missing, MD5 checksum will most probably be wrong!\n", name)
			continue
		}
		// StringListValue and BinaryListValue is currently not implemented
		for _, valueKey := range [...]string{"StringValue", "BinaryValue"} {

			value := req.FormValue(fmt.Sprintf("MessageAttributes.entry.%d.Value.%s", i, valueKey))
			if value != "" {
				attributes = append(attributes, SnsMessageAttribute{Name: name, Value: value, Type: dataType})
				continue
			}
		}
		log.Warnf("StringValue or BinaryValue of MessageAttribute %s is missing, MD5 checksum will most probably be wrong!\n", name)
	}

	return attributes
}

/*** List Topics Response */
type TopicArnResult struct {
	TopicArn string `xml:"TopicArn"`
}

type TopicNamestype struct {
	Member []TopicArnResult `xml:"member"`
}

type ListTopicsResult struct {
	Topics TopicNamestype `xml:"Topics"`
}

type ListTopicsResponse struct {
	Xmlns    string                  `xml:"xmlns,attr"`
	Result   ListTopicsResult        `xml:"ListTopicsResult"`
	Metadata common.ResponseMetadata `xml:"ResponseMetadata"`
}

func NewListTopicsResponse(result ListTopicsResult) *ListTopicsResponse {
	uuid, _ := common.NewUUID()
	return &ListTopicsResponse{
		Xmlns:    "http://queue.amazonaws.com/doc/2012-11-05/",
		Metadata: common.ResponseMetadata{RequestId: uuid},
		Result:   result}
}

/*** Create Topic Response */
type CreateTopicResult struct {
	TopicArn string `xml:"TopicArn"`
}

type CreateTopicResponse struct {
	Xmlns    string                  `xml:"xmlns,attr"`
	Result   CreateTopicResult       `xml:"CreateTopicResult"`
	Metadata common.ResponseMetadata `xml:"ResponseMetadata"`
}

func NewCreateTopicResponse(result CreateTopicResult) *CreateTopicResponse {
	uuid, _ := common.NewUUID()
	return &CreateTopicResponse{
		Xmlns:    "http://queue.amazonaws.com/doc/2012-11-05/",
		Metadata: common.ResponseMetadata{RequestId: uuid},
		Result:   result}
}

/*** Create Subscription ***/
type SubscribeResult struct {
	SubscriptionArn string `xml:"SubscriptionArn"`
}

type SubscribeResponse struct {
	Xmlns    string                  `xml:"xmlns,attr"`
	Result   SubscribeResult         `xml:"SubscribeResult"`
	Metadata common.ResponseMetadata `xml:"ResponseMetadata"`
}

func NewSubscribeResponse(result SubscribeResult) *SubscribeResponse {
	uuid, _ := common.NewUUID()
	return &SubscribeResponse{
		Xmlns:    "http://queue.amazonaws.com/doc/2012-11-05/",
		Metadata: common.ResponseMetadata{RequestId: uuid},
		Result:   result}
}

/***  Set Subscription Response ***/

type SetSubscriptionAttributesResponse struct {
	Xmlns    string                  `xml:"xmlns,attr"`
	Metadata common.ResponseMetadata `xml:"ResponseMetadata"`
}

func NewSetSubscriptionAttributesResponse() *SetSubscriptionAttributesResponse {
	uuid, _ := common.NewUUID()
	return &SetSubscriptionAttributesResponse{
		Xmlns:    "http://queue.amazonaws.com/doc/2012-11-05/",
		Metadata: common.ResponseMetadata{RequestId: uuid}}
}

/*** List Subscriptions Response */
type TopicMemberResult struct {
	TopicArn        string `xml:"TopicArn"`
	Protocol        string `xml:"Protocol"`
	SubscriptionArn string `xml:"SubscriptionArn"`
	Owner           string `xml:"Owner"`
	Endpoint        string `xml:"Endpoint"`
}

type TopicSubscriptions struct {
	Member []TopicMemberResult `xml:"member"`
}

type ListSubscriptionsResult struct {
	Subscriptions TopicSubscriptions `xml:"Subscriptions"`
}

type ListSubscriptionsResponse struct {
	Xmlns    string                  `xml:"xmlns,attr"`
	Result   ListSubscriptionsResult `xml:"ListSubscriptionsResult"`
	Metadata common.ResponseMetadata `xml:"ResponseMetadata"`
}

func NewListSubscriptionsResponse(result ListSubscriptionsResult) *ListSubscriptionsResponse {
	uuid, _ := common.NewUUID()
	return &ListSubscriptionsResponse{
		Xmlns:    "http://queue.amazonaws.com/doc/2012-11-05/",
		Metadata: common.ResponseMetadata{RequestId: uuid},
		Result:   result}
}

/*** List Subscriptions By Topic Response */

type ListSubscriptionsByTopicResult struct {
	Subscriptions TopicSubscriptions `xml:"Subscriptions"`
}

type ListSubscriptionsByTopicResponse struct {
	Xmlns    string                  `xml:"xmlns,attr"`
	Result   ListSubscriptionsResult `xml:"ListSubscriptionsResult"`
	Metadata common.ResponseMetadata `xml:"ResponseMetadata"`
}

func NewListSubscriptionsByTopicResponse(result ListSubscriptionsResult) *ListSubscriptionsByTopicResponse {
	uuid, _ := common.NewUUID()
	return &ListSubscriptionsByTopicResponse{
		Xmlns:    "http://queue.amazonaws.com/doc/2012-11-05/",
		Metadata: common.ResponseMetadata{RequestId: uuid},
		Result:   result}
}

type TopicMessage struct {
	Type              string                `json:"Type,omitempty"`
	MessageId         string                `json:"MessageId,omitempty"`
	TopicArn          string                `json:"TopicArn,omitempty"`
	Subject           string                `json:"Subject,omitempty"`
	Message           string                `json:"Message,omitempty"`
	TimeStamp         string                `json:"TimeStamp,omitempty"`
	MessageAttributes []SnsMessageAttribute `json:"MessageAttributes,omitempty"`
	MessageStructure  string                `json:"MessageStructure,omitempty"`
}

func NewTopicMessage(Type string, TopicArn string, Message string, MessageAttributes []SnsMessageAttribute, MessageStructure string, Subject string) *TopicMessage {
	MessageId, _ := common.NewUUID()
	TimeStamp := fmt.Sprintln(time.Now().Format("2006-01-02T15:04:05:001Z"))
	return &TopicMessage{
		Type:              Type,
		TopicArn:          TopicArn,
		Message:           Message,
		MessageAttributes: MessageAttributes,
		MessageId:         MessageId,
		TimeStamp:         TimeStamp,
		MessageStructure:  MessageStructure,
		Subject:           Subject}
}

func (c *TopicMessage) toString() ([]byte, error) {
	if MessageStructure(c.MessageStructure) == MessageStructureJson {
		message, err := common.ExtractMessageBodyFromJson(c.Message, "sqs")
		if err != nil {
			messageArray := *message
			return []byte(messageArray), nil
		}
		c.Message = *message
	}
	byteMsg, _ := json.Marshal(c)
	return byteMsg, nil
}

/*** Publish ***/

type PublishResult struct {
	MessageId string `xml:"MessageId"`
}

type PublishResponse struct {
	Xmlns    string                  `xml:"xmlns,attr"`
	Result   PublishResult           `xml:"PublishResult"`
	Metadata common.ResponseMetadata `xml:"ResponseMetadata"`
}

func NewPublishResponse(result PublishResult) *PublishResponse {
	uuid, _ := common.NewUUID()
	return &PublishResponse{
		Xmlns:    "http://queue.amazonaws.com/doc/2012-11-05/",
		Metadata: common.ResponseMetadata{RequestId: uuid},
		Result:   result}
}

/*** Unsubscribe ***/
type UnsubscribeResponse struct {
	Xmlns    string                  `xml:"xmlns,attr"`
	Metadata common.ResponseMetadata `xml:"ResponseMetadata"`
}

func NewUnsubscribeResponse() *UnsubscribeResponse {
	uuid, _ := common.NewUUID()
	return &UnsubscribeResponse{
		Xmlns:    "http://queue.amazonaws.com/doc/2012-11-05/",
		Metadata: common.ResponseMetadata{RequestId: uuid}}
}

/*** Delete Topic ***/
type DeleteTopicResponse struct {
	Xmlns    string                  `xml:"xmlns,attr"`
	Metadata common.ResponseMetadata `xml:"ResponseMetadata"`
}

func NewDeleteTopicResponse() *DeleteTopicResponse {
	uuid, _ := common.NewUUID()
	return &DeleteTopicResponse{
		Xmlns:    "http://queue.amazonaws.com/doc/2012-11-05/",
		Metadata: common.ResponseMetadata{RequestId: uuid}}
}

/*** Get Message Attributes ***/
type SnsMessageAttribute struct {
	Name  string `xml:"Name,omitempty"`
	Value string `xml:"Value,omitempty"`
	Type  string `xml:"Type,omitempty"`
}

type GetSnsMessageAttributesResult struct {
	MessageAttrs []SnsMessageAttribute `xml:"MessageAttribute,omitempty"`
}
