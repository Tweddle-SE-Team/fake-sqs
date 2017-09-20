package sns

import (
	"github.com/Tweddle-SE-Team/goaws/backends/common"
)

type SnsErrorType struct {
	HttpError int
	Type      string
	Code      string
	Message   string
}

type Subscription struct {
	TopicArn        string
	Protocol        string
	SubscriptionArn string
	EndPoint        string
	Raw             bool
}

type Topic struct {
	Name          string
	Arn           string
	Subscriptions []*Subscription
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

/*** Create Topic Response */
type CreateTopicResult struct {
	TopicArn string `xml:"TopicArn"`
}

type CreateTopicResponse struct {
	Xmlns    string                  `xml:"xmlns,attr"`
	Result   CreateTopicResult       `xml:"CreateTopicResult"`
	Metadata common.ResponseMetadata `xml:"ResponseMetadata"`
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

/***  Set Subscription Response ***/

type SetSubscriptionAttributesResponse struct {
	Xmlns    string                  `xml:"xmlns,attr"`
	Metadata common.ResponseMetadata `xml:"ResponseMetadata"`
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

/*** List Subscriptions By Topic Response */

type ListSubscriptionsByTopicResult struct {
	Subscriptions TopicSubscriptions `xml:"Subscriptions"`
}

type ListSubscriptionsByTopicResponse struct {
	Xmlns    string                  `xml:"xmlns,attr"`
	Result   ListSubscriptionsResult `xml:"ListSubscriptionsResult"`
	Metadata common.ResponseMetadata `xml:"ResponseMetadata"`
}

type TopicMessage struct {
	Type              string                         `json:"Type,omitempty"`
	MessageId         string                         `json:"MessageId,omitempty"`
	TopicArn          string                         `json:"TopicArn,omitempty"`
	Subject           string                         `json:"Subject,omitempty"`
	Message           string                         `json:"Message,omitempty"`
	TimeStamp         string                         `json:"TimeStamp,omitempty"`
	MessageAttributes map[string]SnsMessageAttribute `json:"MessageAttributes,omitempty"`
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

/*** Unsubscribe ***/
type UnsubscribeResponse struct {
	Xmlns    string                  `xml:"xmlns,attr"`
	Metadata common.ResponseMetadata `xml:"ResponseMetadata"`
}

/*** Delete Topic ***/
type DeleteTopicResponse struct {
	Xmlns    string                  `xml:"xmlns,attr"`
	Metadata common.ResponseMetadata `xml:"ResponseMetadata"`
}

/*** Get Message Attributes ***/
type SnsMessageAttribute struct {
	Name  string `xml:"Name,omitempty"`
	Value string `xml:"Value,omitempty"`
	Type  string `xml:"Type,omitempty"`
}

type GetSnsMessageAttributesResult struct {
	MessageAttrs map[string]SnsMessageAttribute `xml:"MessageAttribute,omitempty"`
}
