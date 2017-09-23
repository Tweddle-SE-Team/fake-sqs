package snsiface

import (
	"github.com/Tweddle-SE-Team/goaws/services/sns"
	"net/http"
)

type SNSAPI interface {
	//AddPermission(*http.Request) (interface{}, string, error)
	//CheckIfPhoneNumberIsOptedOut(*http.Request) (interface{}, string, error)
	//ConfirmSubscription(*http.Request) (interface{}, string, error)
	//CreatePlatformApplication(*http.Request) (interface{}, string, error)
	//CreatePlatformEndpoint(*http.Request) (interface{}, string, error)
	CreateTopic(*http.Request) (interface{}, string, error)
	//DeleteEndpoint(*http.Request) (interface{}, string, error)
	//DeletePlatformApplication(*http.Request) (interface{}, string, error)
	DeleteTopic(*http.Request) (interface{}, string, error)
	//GetEndpointAttributes(*http.Request) (interface{}, string, error)
	//GetPlatformApplicationAttributes(*http.Request) (interface{}, string, error)
	//GetSMSAttributes(*http.Request) (interface{}, string, error)
	//GetSubscriptionAttributes(*http.Request) (interface{}, string, error)
	//GetTopicAttributes(*http.Request) (interface{}, string, error)
	//ListEndpointsByPlatformApplication(*http.Request) (interface{}, string, error)
	//ListPhoneNumbersOptedOut(*http.Request) (interface{}, string, error)
	//ListPlatformApplications(*http.Request) (interface{}, string, error)
	ListSubscriptions(*http.Request) (interface{}, string, error)
	ListSubscriptionsByTopic(*http.Request) (interface{}, string, error)
	ListTopics(*http.Request) (interface{}, string, error)
	//OptInPhoneNumber(*http.Request) (interface{}, string, error)
	Publish(*http.Request) (interface{}, string, error)
	//RemovePermission(*http.Request) (interface{}, string, error)
	//SetEndpointAttributes(*http.Request) (interface{}, string, error)
	//SetPlatformApplicationAttributes(*http.Request) (interface{}, string, error)
	//SetSMSAttributes(*http.Request) (interface{}, string, error)
	SetSubscriptionAttributes(*http.Request) (interface{}, string, error)
	//SetTopicAttributes(*http.Request) (interface{}, string, error)
	Subscribe(*http.Request) (interface{}, string, error)
	Unsubscribe(*http.Request) (interface{}, string, error)
}

var _ SNSAPI = (*sns.SNS)(nil)
