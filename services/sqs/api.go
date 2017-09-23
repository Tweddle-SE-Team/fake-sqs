package sqs

import (
	"errors"
	"fmt"
	"github.com/Tweddle-SE-Team/goaws/services/common"
	log "github.com/sirupsen/logrus"
	"net/http"
	"strconv"
	"time"
)

func (c *SQS) CreateQueue(request *http.Request) (interface{}, string, error) {
	queue := NewQueue(request.FormValue("QueueName"), request.Host)
	queueEquals := func(src interface{}, value interface{}) bool {
		return src.(*Queue).Name == value.(*Queue).Name
	}
	if c.Queues.Get(queue, queueEquals) != nil {
		c.Queues.Put(queue)
	}
	return NewCreateQueueResponse(CreateQueueResult{QueueUrl: queue.URL}), "XML", nil
}

func (c *SQS) DeleteMessage(request *http.Request) (interface{}, string, error) {
	receiptHandle := request.FormValue("ReceiptHandle")
	queueName := GetQueueNameFromRequest(request)
	queue := NewQueue(queueName, request.Host)
	messageEquals := func(src interface{}, value interface{}) bool {
		return src.(Message).ReceiptHandle == value.(string)
	}
	queueEquals := func(src interface{}, value interface{}) bool {
		return src.(*Queue).Name == value.(*Queue).Name
	}
	if q := c.Queues.Get(&queue, queueEquals); q != nil {
		if queue.Messages.Remove(&receiptHandle, messageEquals) {
			return NewDeleteMessageResponse(), "XML", nil
		}
	}
	return nil, "XML", errors.New("MessageDoesNotExist")
}

func (c *SQS) DeleteMessageBatch(request *http.Request) (interface{}, string, error) {
	queueName := GetQueueNameFromRequest(request)
	queue := NewQueue(queueName, request.Host)
	deletedResultEntries := []DeleteMessageBatchResultEntry{}
	notFoundEntries := []BatchResultErrorEntry{}
	queueEquals := func(src interface{}, value interface{}) bool {
		return src.(*Queue).Name == value.(*Queue).Name
	}
	messageEquals := func(src interface{}, value interface{}) bool {
		return src.(*Message).ReceiptHandle == value.(*DeleteEntry).ReceiptHandle
	}
	for i := 2; true; i++ {
		messageId := request.FormValue(fmt.Sprintf("DeleteMessageBatchRequestEntry.%d.Id", i))
		receiptHandle := request.FormValue(fmt.Sprintf("DeleteMessageBatchRequestEntry.%d.ReceiptHandle", i))
		if messageId != "" && receiptHandle != "" {
			deleteEntry := DeleteEntry{
				Id:            messageId,
				ReceiptHandle: receiptHandle,
				Deleted:       false}
			if q := c.Queues.Get(queue, queueEquals); q != nil {
				if q.(Queue).Messages.Remove(&deleteEntry, messageEquals) {
					deleteEntry.Deleted = true
				}
			}
			if deleteEntry.Deleted == true {
				deletedResultEntries = append(deletedResultEntries, DeleteMessageBatchResultEntry{Id: deleteEntry.Id})
			} else {
				notFoundEntries = append(notFoundEntries,
					BatchResultErrorEntry{
						Code:        "1",
						Id:          deleteEntry.Id,
						Message:     "Message not found",
						SenderFault: true})
			}
		} else {
			break
		}
	}

	return NewDeleteMessageBatchResponse(
		DeleteMessageBatchResult{
			Entry: deletedResultEntries,
			Error: notFoundEntries}), "XML", nil
}

func (c *SQS) DeleteQueue(request *http.Request) (interface{}, string, error) {
	queueName := GetQueueNameFromRequest(request)
	queueEquals := func(s interface{}, v interface{}) bool {
		src := s.(*Queue)
		value := v.(*string)
		return src.Name == *value
	}
	c.Queues.Remove(&queueName, queueEquals)
	return NewDeleteQueueResponse(), "XML", nil
}

func (c *SQS) GetQueueAttributes(request *http.Request) (interface{}, string, error) {
	queueName := GetQueueNameFromRequest(request)
	queueEquals := func(s interface{}, v interface{}) bool {
		src := s.(*Queue)
		value := v.(*string)
		return src.Name == *value
	}
	if q := c.Queues.Get(&queueName, queueEquals); q != nil {
		result := GetQueueAttributesResult{Attrs: []Attribute{
			Attribute{Name: "VisibilityTimeout", Value: strconv.Itoa(q.(*Queue).TimeoutSecs)},
			Attribute{Name: "DelaySeconds", Value: "0"},
			Attribute{Name: "ReceiveMessageWaitTimeSeconds", Value: "0"},
			Attribute{Name: "ApproximateNumberOfMessages", Value: strconv.Itoa(q.(*Queue).Messages.Size())},
			Attribute{Name: "ApproximateNumberOfMessagesNotVisible", Value: "0"},
			Attribute{Name: "CreatedTimestamp", Value: "0000000000"},
			Attribute{Name: "LastModifiedTimestamp", Value: "0000000000"},
			Attribute{Name: "QueueArn", Value: q.(*Queue).Arn},
		}}
		return NewGetQueueAttributesResponse(result), "XML", nil
	} else {
		return nil, "XML", errors.New("QueueNotFound")
	}
}

func (c *SQS) GetQueueUrl(request *http.Request) (interface{}, string, error) {
	queueName := request.FormValue("QueueName")
	queueEquals := func(s interface{}, v interface{}) bool {
		src := s.(*Queue)
		value := v.(*string)
		return src.Name == *value
	}
	if q := c.Queues.Get(&queueName, queueEquals); q != nil {
		return NewGetQueueUrlResponse(GetQueueUrlResult{QueueUrl: q.(*Queue).URL}), "XML", nil
	} else {
		return nil, "XML", errors.New("QueueNotFound")
	}
}

func (c *SQS) ListQueues(request *http.Request) (interface{}, string, error) {
	queueUrls := make([]string, 0)
	for q := range c.Queues.Iterator() {
		queue := q.(*Queue)
		queueUrls = append(queueUrls, queue.URL)
	}
	return NewListQueuesResponse(ListQueuesResult{QueueUrl: queueUrls}), "XML", nil
}

func (c *SQS) PurgeQueue(request *http.Request) (interface{}, string, error) {
	queueName := GetQueueNameFromRequest(request)
	queueEquals := func(s interface{}, v interface{}) bool {
		src := s.(*Queue)
		value := v.(*string)
		return src.Name == *value
	}
	if q := c.Queues.Get(&queueName, queueEquals); q != nil {
		q.(*Queue).Messages.Empty()
		return NewPurgeQueueResponse(), "XML", nil
	} else {
		return nil, "XML", errors.New("QueueNotFound")
	}
}

func (c *SQS) ReceiveMessage(request *http.Request) (interface{}, string, error) {
	receiveParameters := map[string]int{
		"waitTimeSeconds":     0,
		"maxNumberOfMessages": 1}
	for key, _ := range receiveParameters {
		if param := request.FormValue(key); param != "" {
			receiveParameters[key], _ = strconv.Atoi(param)
		}
	}
	queueName := GetQueueNameFromRequest(request)
	queueEquals := func(s interface{}, v interface{}) bool {
		src := s.(*Queue)
		value := v.(*string)
		return src.Name == *value
	}
	messageEquals := func(src interface{}, value interface{}) bool {
		return src.(*Message).ReceiptHandle == value.(*Message).ReceiptHandle
	}
	log.Warnf("Queue Name %+v", queueName)
	queue := c.Queues.Get(&queueName, queueEquals)
	if queue == nil {
		return nil, "XML", errors.New("QueueNotFound")
	}
	time.Sleep(1000 * time.Millisecond * time.Duration(receiveParameters["waitTimeSeconds"]))
	var resultMessages []*Message
	if queue.(*Queue).Messages.Size() > 0 {
		for m := range queue.(*Queue).Messages.Iterator() {
			if receiveParameters["maxNumberOfMessages"] <= 0 {
				break
			} else {
				receiveParameters["maxNumberOfMessages"]--
			}
			message := m.(*Message)
			message.UpdateReceiptHandle()
			resultMessages = append(resultMessages, message)
			queue.(*Queue).Messages.Remove(message, messageEquals)
		}
		return NewReceiveMessageResponse(ReceiveMessageResult{Message: resultMessages}), "XML", nil
	} else {
		return NewReceiveMessageResponse(ReceiveMessageResult{}), "XML", nil
	}
}

func (c *SQS) SendMessage(request *http.Request) (interface{}, string, error) {
	messageBody := request.FormValue("MessageBody")
	messageAttributes, md5OfMessageAttributes := c.ExtractSqsMessageAttributes(request)
	queueName := GetQueueNameFromRequest(request)
	queueEquals := func(s interface{}, v interface{}) bool {
		src := s.(*Queue)
		value := v.(*string)
		return src.Name == *value
	}
	q := c.Queues.Get(&queueName, queueEquals)
	if q == nil {
		return nil, "XML", errors.New("QueueNotFound")
	}
	var messageAttrs []SqsMessageAttribute
	for k := range messageAttributes {
		messageAttrs = append(messageAttrs, messageAttributes[k])
	}
	message := NewMessage([]byte(messageBody), messageAttrs, "", md5OfMessageAttributes)
	q.(*Queue).Messages.Put(message)
	return NewSendMessageResponse(
		SendMessageResult{
			MD5OfMessageAttributes: message.MD5OfMessageAttributes,
			MD5OfMessageBody:       message.MD5OfMessageBody,
			MessageId:              message.MessageId}), "XML", nil
}

func (c *SQS) SetQueueAttributes(request *http.Request) (interface{}, string, error) {
	return NewSetQueueAttributesResponse(), "XML", nil
}

type DeleteEntry struct {
	Id            string
	ReceiptHandle string
	Error         string
	Deleted       bool
}

/*** List Queues Response */
type ListQueuesResult struct {
	QueueUrl []string `xml:"QueueUrl"`
}

type ListQueuesResponse struct {
	Xmlns    string                  `xml:"xmlns,attr"`
	Result   ListQueuesResult        `xml:"ListQueuesResult"`
	Metadata common.ResponseMetadata `xml:"ResponseMetadata"`
}

func NewListQueuesResponse(Result ListQueuesResult) *ListQueuesResponse {
	return &ListQueuesResponse{
		Xmlns:    "http://queue.amazonaws.com/doc/2012-11-05/",
		Metadata: common.ResponseMetadata{RequestId: "00000000-0000-0000-0000-000000000000"},
		Result:   Result}
}

/*** Create Queue Response */
type CreateQueueResult struct {
	QueueUrl string `xml:"QueueUrl"`
}

type CreateQueueResponse struct {
	Xmlns    string                  `xml:"xmlns,attr"`
	Result   CreateQueueResult       `xml:"CreateQueueResult"`
	Metadata common.ResponseMetadata `xml:"ResponseMetadata"`
}

func NewCreateQueueResponse(Result CreateQueueResult) *CreateQueueResponse {
	return &CreateQueueResponse{
		Xmlns:    "http://queue.amazonaws.com/doc/2012-11-05/",
		Metadata: common.ResponseMetadata{RequestId: "00000000-0000-0000-0000-000000000000"},
		Result:   Result}
}

/*** Send Message Response */

type SendMessageResult struct {
	MD5OfMessageAttributes string `xml:"MD5OfMessageAttributes"`
	MD5OfMessageBody       string `xml:"MD5OfMessageBody"`
	MessageId              string `xml:"MessageId"`
}

type SendMessageResponse struct {
	Xmlns    string                  `xml:"xmlns,attr"`
	Result   SendMessageResult       `xml:"SendMessageResult"`
	Metadata common.ResponseMetadata `xml:"ResponseMetadata"`
}

func NewSendMessageResponse(Result SendMessageResult) *SendMessageResponse {
	return &SendMessageResponse{
		Xmlns:    "http://queue.amazonaws.com/doc/2012-11-05/",
		Metadata: common.ResponseMetadata{RequestId: "00000000-0000-0000-0000-000000000000"},
		Result:   Result}
}

/*** Receive Message Response */
type ReceiveMessageResult struct {
	Message []*Message `xml:"Message,omitempty"`
}

type ReceiveMessageResponse struct {
	Xmlns    string                  `xml:"xmlns,attr"`
	Result   ReceiveMessageResult    `xml:"ReceiveMessageResult"`
	Metadata common.ResponseMetadata `xml:"ResponseMetadata"`
}

func NewReceiveMessageResponse(Result ReceiveMessageResult) *ReceiveMessageResponse {
	return &ReceiveMessageResponse{
		Xmlns:    "http://queue.amazonaws.com/doc/2012-11-05/",
		Metadata: common.ResponseMetadata{RequestId: "00000000-0000-0000-0000-000000000000"},
		Result:   Result}
}

/*** Delete Message Response */
type DeleteMessageResponse struct {
	Xmlns    string                  `xml:"xmlns,attr,omitempty"`
	Metadata common.ResponseMetadata `xml:"ResponseMetadata,omitempty"`
}

type DeleteQueueResponse struct {
	Xmlns    string                  `xml:"xmlns,attr,omitempty"`
	Metadata common.ResponseMetadata `xml:"ResponseMetadata,omitempty"`
}

func NewDeleteQueueResponse() *DeleteQueueResponse {
	return &DeleteQueueResponse{
		Xmlns:    "http://queue.amazonaws.com/doc/2012-11-05/",
		Metadata: common.ResponseMetadata{RequestId: "00000000-0000-0000-0000-000000000000"}}
}

func NewDeleteMessageResponse() *DeleteMessageResponse {
	return &DeleteMessageResponse{
		Xmlns:    "http://queue.amazonaws.com/doc/2012-11-05/",
		Metadata: common.ResponseMetadata{RequestId: "00000000-0000-0000-0000-000000000000"}}
}

type DeleteMessageBatchResultEntry struct {
	Id string `xml:"Id"`
}

type BatchResultErrorEntry struct {
	Code        string `xml:"Code"`
	Id          string `xml:"Id"`
	Message     string `xml:"Message,omitempty"`
	SenderFault bool   `xml:"SenderFault"`
}

type DeleteMessageBatchResult struct {
	Entry []DeleteMessageBatchResultEntry `xml:"DeleteMessageBatchResultEntry"`
	Error []BatchResultErrorEntry         `xml:"BatchResultErrorEntry,omitempty"`
}

/*** Delete Message Batch Response */
type DeleteMessageBatchResponse struct {
	Xmlns    string                   `xml:"xmlns,attr,omitempty"`
	Result   DeleteMessageBatchResult `xml:"DeleteMessageBatchResult"`
	Metadata common.ResponseMetadata  `xml:"ResponseMetadata,omitempty"`
}

func NewDeleteMessageBatchResponse(Result DeleteMessageBatchResult) *DeleteMessageBatchResponse {
	return &DeleteMessageBatchResponse{
		Xmlns:    "http://queue.amazonaws.com/doc/2012-11-05/",
		Metadata: common.ResponseMetadata{RequestId: "00000000-0000-0000-0000-000000000000"},
		Result:   Result}
}

/*** Purge Queue Response */
type PurgeQueueResponse struct {
	Xmlns    string                  `xml:"xmlns,attr,omitempty"`
	Metadata common.ResponseMetadata `xml:"ResponseMetadata,omitempty"`
}

func NewPurgeQueueResponse() *PurgeQueueResponse {
	return &PurgeQueueResponse{
		Xmlns:    "http://queue.amazonaws.com/doc/2012-11-05/",
		Metadata: common.ResponseMetadata{RequestId: "00000000-0000-0000-0000-000000000000"}}
}

/*** Get Queue Url Response */
type GetQueueUrlResult struct {
	QueueUrl string `xml:"QueueUrl,omitempty"`
}

type GetQueueUrlResponse struct {
	Xmlns    string                  `xml:"xmlns,attr,omitempty"`
	Result   GetQueueUrlResult       `xml:"GetQueueUrlResult"`
	Metadata common.ResponseMetadata `xml:"ResponseMetadata,omitempty"`
}

func NewGetQueueUrlResponse(Result GetQueueUrlResult) *GetQueueUrlResponse {
	return &GetQueueUrlResponse{
		Xmlns:    "http://queue.amazonaws.com/doc/2012-11-05/",
		Metadata: common.ResponseMetadata{RequestId: "00000000-0000-0000-0000-000000000000"},
		Result:   Result}
}

/*** Get Queue Attributes ***/
type Attribute struct {
	Name  string `xml:"Name,omitempty"`
	Value string `xml:"Value,omitempty"`
}

type GetQueueAttributesResult struct {
	/* VisibilityTimeout, DelaySeconds, ReceiveMessageWaitTimeSeconds, ApproximateNumberOfMessages
	   ApproximateNumberOfMessagesNotVisible, CreatedTimestamp, LastModifiedTimestamp, QueueArn */
	Attrs []Attribute `xml:"Attribute,omitempty"`
}

type GetQueueAttributesResponse struct {
	Xmlns    string                   `xml:"xmlns,attr,omitempty"`
	Result   GetQueueAttributesResult `xml:"GetQueueAttributesResult"`
	Metadata common.ResponseMetadata  `xml:"ResponseMetadata,omitempty"`
}

func NewGetQueueAttributesResponse(Result GetQueueAttributesResult) *GetQueueAttributesResponse {
	return &GetQueueAttributesResponse{
		Xmlns:    "http://queue.amazonaws.com/doc/2012-11-05/",
		Metadata: common.ResponseMetadata{RequestId: "00000000-0000-0000-0000-000000000000"},
		Result:   Result}
}

type SetQueueAttributesResponse struct {
	Xmlns    string                  `xml:"xmlns,attr,omitempty"`
	Metadata common.ResponseMetadata `xml:"ResponseMetadata,omitempty"`
}

func NewSetQueueAttributesResponse() *SetQueueAttributesResponse {
	return &SetQueueAttributesResponse{
		Xmlns:    "http://queue.amazonaws.com/doc/2012-11-05/",
		Metadata: common.ResponseMetadata{RequestId: "00000000-0000-0000-0000-000000000000"}}
}

/*** Get Message Attributes ***/
type SqsMessageAttribute struct {
	Name  string                   `xml:"Name,omitempty"`
	Value SqsMessageAttributeValue `xml:"Value,omitempty"`
}

type SqsMessageAttributeValue struct {
	StringValue     string `xml:"StringValue,omitempty" json:"StringValue,omitempty"`
	BinaryValue     string `xml:"BinaryValue,omitempty" json:"BinaryValue,omitempty"`
	BinaryListValue string `xml:"BinaryListValue,omitempty" json:"BinaryListValue,omitempty"`
	StringListValue string `xml:"StringListValue,omitempty" json:"StringListValue,omitempty"`
	DataType        string `xml:"DataType,omitempty"`
}

type GetSqsMessageAttributesResult struct {
	MessageAttrs []SqsMessageAttribute `xml:"MessageAttribute,omitempty"`
}
