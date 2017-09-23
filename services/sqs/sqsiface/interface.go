package sqsiface

import (
	"github.com/Tweddle-SE-Team/goaws/services/sqs"
	"net/http"
)

type SQSAPI interface {
	//AddPermission(*http.Request) (interface{}, string, error)
	//ChangeMessageVisibility(*http.Request) (interface{}, string, error)
	//ChangeMessageVisibilityBatch(*http.Request) (interface{}, string, error)
	CreateQueue(*http.Request) (interface{}, string, error)
	DeleteMessage(*http.Request) (interface{}, string, error)
	DeleteMessageBatch(*http.Request) (interface{}, string, error)
	DeleteQueue(*http.Request) (interface{}, string, error)
	GetQueueAttributes(*http.Request) (interface{}, string, error)
	GetQueueUrl(*http.Request) (interface{}, string, error)
	//ListDeadLetterSourceQueues(*http.Request) (interface{}, string, error)
	ListQueues(*http.Request) (interface{}, string, error)
	PurgeQueue(*http.Request) (interface{}, string, error)
	ReceiveMessage(*http.Request) (interface{}, string, error)
	//RemovePermission(*http.Request) (interface{}, string, error)
	SendMessage(*http.Request) (interface{}, string, error)
	//SendMessageBatch(*http.Request) (interface{}, string, error)
	SetQueueAttributes(*http.Request) (interface{}, string, error)
}

var _ SQSAPI = (*sqs.SQS)(nil)
