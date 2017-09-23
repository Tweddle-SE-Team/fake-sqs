package sns

import (
	"github.com/Tweddle-SE-Team/goaws/services/common"
	"net/http"
)

var Errors = common.ErrorHandler{
	"TopicNotFound": common.ErrorType{
		HttpError: http.StatusBadRequest,
		Type:      "Not Found",
		Code:      "AWS.SimpleNotificationService.NonExistentTopic",
		Message:   "The specified topic does not exist for this wsdl version."},
	"SubscriptionNotFound": common.ErrorType{
		HttpError: http.StatusBadRequest,
		Type:      "Not Found",
		Code:      "AWS.SimpleNotificationService.NonExistentSubscription",
		Message:   "The specified subscription does not exist for this wsdl version."},
	"TopicExists": common.ErrorType{
		HttpError: http.StatusBadRequest,
		Type:      "Duplicate",
		Code:      "AWS.SimpleNotificationService.TopicAlreadyExists",
		Message:   "The specified topic already exists."}}
