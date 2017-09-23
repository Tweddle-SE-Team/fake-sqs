package router

import (
	"io"
	"net/http"

	"encoding/json"
	"encoding/xml"
	log "github.com/sirupsen/logrus"

	"github.com/Tweddle-SE-Team/goaws/services/common"
	"github.com/Tweddle-SE-Team/goaws/services/sns"
	"github.com/Tweddle-SE-Team/goaws/services/sqs"
	"github.com/gorilla/mux"
)

type AWSHandler func(request *http.Request) (interface{}, string, error)

// New returns a new router
func New() http.Handler {
	r := mux.NewRouter()

	r.HandleFunc("/", actionHandler).Methods("GET", "POST")
	r.HandleFunc("/queue/{queueName}", actionHandler).Methods("GET", "POST")

	return r
}

var errorHandlers = []common.ErrorHandler{
	sns.Errors,
	sqs.Errors}

func selectErrorHandler(err error) common.ErrorType {
	errorMessage := err.Error()
	errorType := common.ErrorType{}
	for _, errorHandler := range errorHandlers {
		for e, _ := range errorHandler {
			if e == errorMessage {
				return errorHandler[e]
			}
			errorType = errorHandler[e]
		}
	}
	return errorType
}

func createErrorResponse(writer http.ResponseWriter, request *http.Request, err error) {
	e := selectErrorHandler(err)
	response := common.ErrorResponse{
		common.ErrorResult{
			Type:      e.Type,
			Code:      e.Code,
			Message:   e.Message,
			RequestId: "00000000-0000-0000-0000-000000000000"}}

	writer.WriteHeader(e.HttpError)
	encoded := xml.NewEncoder(writer)
	encoded.Indent("  ", "    ")
	if err := encoded.Encode(response); err != nil {
		log.Printf("error: %v\n", err)
	}
}

func sendResponse(writer http.ResponseWriter, request *http.Request, response interface{}, content string) {
	if content == "JSON" {
		writer.Header().Set("Content-Type", "application/json")
	} else {
		writer.Header().Set("Content-Type", "application/xml")
	}
	if content == "JSON" {
		encoded := json.NewEncoder(writer)
		if err := encoded.Encode(response); err != nil {
			log.Printf("error: %v\n", err)
		}
	} else {
		encoded := xml.NewEncoder(writer)
		encoded.Indent("  ", "    ")
		if err := encoded.Encode(response); err != nil {
			log.Printf("error: %v\n", err)
		}
	}
}

func response(writer http.ResponseWriter, request *http.Request) {
	fn, ok := routingTable[request.FormValue("Action")]
	if !ok {
		log.Println("Bad Request - Action:", request.FormValue("Action"))
		writer.WriteHeader(http.StatusBadRequest)
		io.WriteString(writer, "Bad Request")
		return
	}
	output, content, err := fn(request)
	if err != nil {
		createErrorResponse(writer, request, err)
	} else {
		sendResponse(writer, request, output, content)
	}
}

var routingTable = map[string]AWSHandler{
	// SQS
	"ListQueues":         sqs.Service.ListQueues,
	"CreateQueue":        sqs.Service.CreateQueue,
	"GetQueueAttributes": sqs.Service.GetQueueAttributes,
	"SetQueueAttributes": sqs.Service.SetQueueAttributes,
	"SendMessage":        sqs.Service.SendMessage,
	"ReceiveMessage":     sqs.Service.ReceiveMessage,
	"DeleteMessage":      sqs.Service.DeleteMessage,
	"DeleteMessageBatch": sqs.Service.DeleteMessageBatch,
	"GetQueueUrl":        sqs.Service.GetQueueUrl,
	"PurgeQueue":         sqs.Service.PurgeQueue,
	"DeleteQueue":        sqs.Service.DeleteQueue,

	// SNS
	"ListTopics":                sns.Service.ListTopics,
	"CreateTopic":               sns.Service.CreateTopic,
	"DeleteTopic":               sns.Service.DeleteTopic,
	"Subscribe":                 sns.Service.Subscribe,
	"SetSubscriptionAttributes": sns.Service.SetSubscriptionAttributes,
	"ListSubscriptionsByTopic":  sns.Service.ListSubscriptionsByTopic,
	"ListSubscriptions":         sns.Service.ListSubscriptions,
	"Unsubscribe":               sns.Service.Unsubscribe,
	"Publish":                   sns.Service.Publish,
}

func actionHandler(writer http.ResponseWriter, request *http.Request) {
	http.HandlerFunc(response).ServeHTTP(writer, request)
}
