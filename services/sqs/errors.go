package sqs

import (
	"github.com/Tweddle-SE-Team/goaws/services/common"
	"net/http"
)

var Errors = map[string]common.ErrorType{
	"QueueNotFound": common.ErrorType{
		HttpError: http.StatusBadRequest,
		Type:      "Not Found",
		Code:      "AWS.SimpleQueueService.NonExistentQueue",
		Message:   "The specified queue does not exist for this wsdl version."},
	"QueueExists": common.ErrorType{
		HttpError: http.StatusBadRequest,
		Type:      "Duplicate",
		Code:      "AWS.SimpleQueueService.QueueExists",
		Message:   "The specified queue already exists."},
	"MessageDoesNotExist": common.ErrorType{
		HttpError: http.StatusNotFound,
		Type:      "Not Found",
		Code:      "AWS.SimpleQueueService.QueueExists",
		Message:   "The specified queue does not contain the message specified."},
	"GeneralError": common.ErrorType{
		HttpError: http.StatusBadRequest,
		Type:      "GeneralError",
		Code:      "AWS.SimpleQueueService.GeneralError",
		Message:   "General Error."}}
