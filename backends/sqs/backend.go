package sqs

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/Tweddle-SE-Team/goaws/backends/common"
	"github.com/gorilla/mux"
)

var SqsErrors map[string]SqsErrorType

var SyncQueues = struct {
	sync.RWMutex
	Queues map[string]*Queue
}{Queues: make(map[string]*Queue)}

//var Queues map[string]*Queue

func init() {
	SyncQueues.Queues = make(map[string]*Queue)

	SqsErrors = make(map[string]SqsErrorType)
	err1 := SqsErrorType{HttpError: http.StatusBadRequest, Type: "Not Found", Code: "AWS.SimpleQueueService.NonExistentQueue", Message: "The specified queue does not exist for this wsdl version."}
	SqsErrors["QueueNotFound"] = err1
	err2 := SqsErrorType{HttpError: http.StatusBadRequest, Type: "Duplicate", Code: "AWS.SimpleQueueService.QueueExists", Message: "The specified queue already exists."}
	SqsErrors["QueueExists"] = err2
	err3 := SqsErrorType{HttpError: http.StatusNotFound, Type: "Not Found", Code: "AWS.SimpleQueueService.QueueExists", Message: "The specified queue does not contain the message specified."}
	SqsErrors["MessageDoesNotExist"] = err3
	err4 := SqsErrorType{HttpError: http.StatusBadRequest, Type: "GeneralError", Code: "AWS.SimpleQueueService.GeneralError", Message: "General Error."}
	SqsErrors["GeneralError"] = err4
}

func ListQueues(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/xml")
	respStruct := ListQueuesResponse{}
	respStruct.Xmlns = "http://queue.amazonaws.com/doc/2012-11-05/"
	respStruct.Metadata = common.ResponseMetadata{RequestId: "00000000-0000-0000-0000-000000000000"}
	respStruct.Result.QueueUrl = make([]string, 0)

	log.Println("Listing Queues")
	for _, queue := range SyncQueues.Queues {
		SyncQueues.Lock()
		respStruct.Result.QueueUrl = append(respStruct.Result.QueueUrl, queue.URL)
		SyncQueues.Unlock()
	}
	enc := xml.NewEncoder(w)
	enc.Indent("  ", "    ")
	if err := enc.Encode(respStruct); err != nil {
		log.Printf("error: %v\n", err)
	}
}

func CreateQueue(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/xml")
	queueName := req.FormValue("QueueName")
	queueUrl := "http://" + req.Host + "/queue/" + queueName

	if _, ok := SyncQueues.Queues[queueName]; !ok {
		log.Println("Creating Queue:", queueName)
		queue := &Queue{Name: queueName, URL: queueUrl, Arn: queueUrl, TimeoutSecs: 30}
		SyncQueues.RLock()
		SyncQueues.Queues[queueName] = queue
		SyncQueues.RUnlock()
	}

	respStruct := CreateQueueResponse{"http://queue.amazonaws.com/doc/2012-11-05/", CreateQueueResult{QueueUrl: queueUrl}, common.ResponseMetadata{RequestId: "00000000-0000-0000-0000-000000000000"}}
	enc := xml.NewEncoder(w)
	enc.Indent("  ", "    ")
	if err := enc.Encode(respStruct); err != nil {
		log.Printf("error: %v\n", err)
	}
}

func SendMessage(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/xml")
	messageBody := req.FormValue("MessageBody")
	messageAttributes := common.ExtractMessageAttributes(req)
	log.Printf("message attributes %v", messageAttributes)

	queueUrl := getQueueFromPath(req.FormValue("QueueUrl"), req.URL.String())

	queueName := ""
	if queueUrl == "" {
		vars := mux.Vars(req)
		queueName = vars["queueName"]
	} else {
		uriSegments := strings.Split(queueUrl, "/")
		queueName = uriSegments[len(uriSegments)-1]
	}

	if _, ok := SyncQueues.Queues[queueName]; !ok {
		// Queue does not exists
		createErrorResponse(w, req, "QueueNotFound")
		return
	}

	log.Println("Putting Message in Queue:", queueName)
	var messageAttrs []common.MessageAttribute
	for k := range messageAttributes {
		messageAttrs = append(messageAttrs, messageAttributes[k])
	}
	msg := Message{MessageBody: []byte(messageBody), MessageAttributes: messageAttrs}
	msg.MD5OfMessageBody = common.GetMD5Hash(messageBody)
	msg.MD5OfMessageAttributes = common.HashAttributes(messageAttributes)
	msg.Uuid, _ = common.NewUUID()
	SyncQueues.Lock()
	SyncQueues.Queues[queueName].Messages = append(SyncQueues.Queues[queueName].Messages, msg)
	SyncQueues.Unlock()
	common.LogMessage(fmt.Sprintf("%s: Queue: %s, Message: %s\n", time.Now().Format("2006-01-02 15:04:05"), queueName, msg.MessageBody))

	respStruct := SendMessageResponse{"http://queue.amazonaws.com/doc/2012-11-05/", SendMessageResult{MD5OfMessageAttributes: msg.MD5OfMessageAttributes, MD5OfMessageBody: msg.MD5OfMessageBody, MessageId: msg.Uuid}, common.ResponseMetadata{RequestId: "00000000-0000-0000-0000-000000000000"}}
	enc := xml.NewEncoder(w)
	enc.Indent("  ", "    ")
	if err := enc.Encode(respStruct); err != nil {
		log.Printf("error: %v\n", err)
	}
}

func ReceiveMessage(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/xml")

	waitTimeSeconds := 0
	wts := req.FormValue("WaitTimeSeconds")
	if wts != "" {
		waitTimeSeconds, _ = strconv.Atoi(wts)
	}
	maxNumberOfMessages := 1
	mom := req.FormValue("MaxNumberOfMessages")
	if mom != "" {
		maxNumberOfMessages, _ = strconv.Atoi(mom)
	}

	queueUrl := getQueueFromPath(req.FormValue("QueueUrl"), req.URL.String())

	queueName := ""
	if queueUrl == "" {
		vars := mux.Vars(req)
		queueName = vars["queueName"]
	} else {
		uriSegments := strings.Split(queueUrl, "/")
		queueName = uriSegments[len(uriSegments)-1]
	}

	if _, ok := SyncQueues.Queues[queueName]; !ok {
		createErrorResponse(w, req, "QueueNotFound")
		return
	}

	var message []*ResultMessage
	respStruct := ReceiveMessageResponse{}

	loops := waitTimeSeconds * 10
	for len(SyncQueues.Queues[queueName].Messages)-numberOfHiddenMessagesInQueue(*SyncQueues.Queues[queueName]) == 0 && loops > 0 {
		time.Sleep(100 * time.Millisecond)
		loops--
	}
	log.Println("Getting Message from Queue:", queueName)

	if len(SyncQueues.Queues[queueName].Messages) > 0 {
		numMsg := 0
		message = make([]*ResultMessage, 0)
		for i := range SyncQueues.Queues[queueName].Messages {
			if numMsg >= maxNumberOfMessages {
				break
			}
			timeout := time.Now().Add(time.Duration(-SyncQueues.Queues[queueName].TimeoutSecs) * time.Second)
			if (SyncQueues.Queues[queueName].Messages[i].ReceiptHandle != "") && (timeout.Before(SyncQueues.Queues[queueName].Messages[i].ReceiptTime)) {
				continue
			} else {
				SyncQueues.Lock() // Lock the Queues
				uuid, _ := common.NewUUID()
				SyncQueues.Queues[queueName].Messages[i].ReceiptHandle = SyncQueues.Queues[queueName].Messages[i].Uuid + "#" + uuid
				SyncQueues.Queues[queueName].Messages[i].ReceiptTime = time.Now()
				message = append(message, &ResultMessage{})
				message[numMsg].MessageId = SyncQueues.Queues[queueName].Messages[i].Uuid
				for k := range SyncQueues.Queues[queueName].Messages[i].MessageAttributes {
					message[numMsg].MessageAttributes = append(message[numMsg].MessageAttributes, SyncQueues.Queues[queueName].Messages[i].MessageAttributes[k])
				}
				message[numMsg].Body = SyncQueues.Queues[queueName].Messages[i].MessageBody
				message[numMsg].ReceiptHandle = SyncQueues.Queues[queueName].Messages[i].ReceiptHandle
				message[numMsg].MD5OfBody = common.GetMD5Hash(string(message[numMsg].Body))
				SyncQueues.Unlock() // Unlock the Queues
				numMsg++
			}
		}

		respStruct = ReceiveMessageResponse{"http://queue.amazonaws.com/doc/2012-11-05/", ReceiveMessageResult{Message: message}, common.ResponseMetadata{RequestId: "00000000-0000-0000-0000-000000000000"}}
	} else {
		log.Println("No messages in Queue:", queueName)
		respStruct = ReceiveMessageResponse{Xmlns: "http://queue.amazonaws.com/doc/2012-11-05/", Result: ReceiveMessageResult{}, Metadata: common.ResponseMetadata{RequestId: "00000000-0000-0000-0000-000000000000"}}
	}
	enc := xml.NewEncoder(w)
	enc.Indent("  ", "    ")
	if err := enc.Encode(respStruct); err != nil {
		log.Printf("error: %v\n", err)
	}
}

func numberOfHiddenMessagesInQueue(queue Queue) int {
	num := 0
	for i := range queue.Messages {
		timeout := time.Now().Add(time.Duration(-queue.TimeoutSecs) * time.Second)
		if (queue.Messages[i].ReceiptHandle != "") && (timeout.Before(queue.Messages[i].ReceiptTime)) {
			num++
		}
	}
	return num
}

type DeleteEntry struct {
	Id            string
	ReceiptHandle string
	Error         string
	Deleted       bool
}

func DeleteMessageBatch(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/xml")
	req.ParseForm()

	queueUrl := getQueueFromPath(req.FormValue("QueueUrl"), req.URL.String())
	queueName := ""
	if queueUrl == "" {
		vars := mux.Vars(req)
		queueName = vars["queueName"]
	} else {
		uriSegments := strings.Split(queueUrl, "/")
		queueName = uriSegments[len(uriSegments)-1]
	}

	deleteEntries := []DeleteEntry{}

	for k, v := range req.Form {
		keySegments := strings.Split(k, ".")
		if keySegments[0] == "DeleteMessageBatchRequestEntry" {
			keyIndex, err := strconv.Atoi(keySegments[1])

			if err != nil {
				createErrorResponse(w, req, "Error")
				return
			}

			if len(deleteEntries) < keyIndex {
				newDeleteEntries := make([]DeleteEntry, keyIndex)
				copy(newDeleteEntries, deleteEntries)
				deleteEntries = newDeleteEntries
			}

			if keySegments[2] == "Id" {
				deleteEntries[keyIndex-1].Id = v[0]
			}

			if keySegments[2] == "ReceiptHandle" {
				deleteEntries[keyIndex-1].ReceiptHandle = v[0]
			}
		}
	}

	deletedEntries := make([]DeleteMessageBatchResultEntry, 0)

	SyncQueues.Lock()
	if _, ok := SyncQueues.Queues[queueName]; ok {
		for i, msg := range SyncQueues.Queues[queueName].Messages {
			for _, deleteEntry := range deleteEntries {
				if msg.ReceiptHandle == deleteEntry.ReceiptHandle {
					SyncQueues.Queues[queueName].Messages = append(SyncQueues.Queues[queueName].Messages[:i], SyncQueues.Queues[queueName].Messages[i+1:]...)

					deleteEntry.Deleted = true
					deletedEntry := DeleteMessageBatchResultEntry{Id: deleteEntry.Id}
					deletedEntries = append(deletedEntries, deletedEntry)
				}
			}
		}
	}
	SyncQueues.Unlock()

	notFoundEntries := make([]BatchResultErrorEntry, 0)
	for _, deleteEntry := range deleteEntries {
		if deleteEntry.Deleted == false {
			notFoundEntries = append(notFoundEntries, BatchResultErrorEntry{
				Code:        "1",
				Id:          deleteEntry.Id,
				Message:     "Message not found",
				SenderFault: true})
		}
	}

	respStruct := DeleteMessageBatchResponse{
		"http://queue.amazonaws.com/doc/2012-11-05/",
		DeleteMessageBatchResult{Entry: deletedEntries, Error: notFoundEntries},
		common.ResponseMetadata{RequestId: "00000000-0000-0000-0000-000000000001"}}

	enc := xml.NewEncoder(w)
	enc.Indent(" ", "    ")
	if err := enc.Encode(respStruct); err != nil {
		log.Printf("error: %v\n", err)
	}
}

func DeleteMessage(w http.ResponseWriter, req *http.Request) {
	// Sent response type
	w.Header().Set("Content-Type", "application/xml")

	// Retrieve FormValues required
	receiptHandle := req.FormValue("ReceiptHandle")

	// Retrieve FormValues required
	queueUrl := getQueueFromPath(req.FormValue("QueueUrl"), req.URL.String())
	queueName := ""
	if queueUrl == "" {
		vars := mux.Vars(req)
		queueName = vars["queueName"]
	} else {
		uriSegments := strings.Split(queueUrl, "/")
		queueName = uriSegments[len(uriSegments)-1]
	}

	log.Println("Deleting Message, Queue:", queueName, ", ReceiptHandle:", receiptHandle)

	// Find queue/message with the receipt handle and delete
	SyncQueues.Lock()
	if _, ok := SyncQueues.Queues[queueName]; ok {
		for i, msg := range SyncQueues.Queues[queueName].Messages {
			if msg.ReceiptHandle == receiptHandle {
				//Delete message from Q
				SyncQueues.Queues[queueName].Messages = append(SyncQueues.Queues[queueName].Messages[:i], SyncQueues.Queues[queueName].Messages[i+1:]...)

				SyncQueues.Unlock()
				// Create, encode/xml and send response
				respStruct := DeleteMessageResponse{"http://queue.amazonaws.com/doc/2012-11-05/", common.ResponseMetadata{RequestId: "00000000-0000-0000-0000-000000000001"}}
				enc := xml.NewEncoder(w)
				enc.Indent("  ", "    ")
				if err := enc.Encode(respStruct); err != nil {
					log.Printf("error: %v\n", err)
				}
				return
			}
		}
		log.Println("Receipt Handle not found")
	} else {
		log.Println("Queue not found")
	}
	SyncQueues.Unlock()

	createErrorResponse(w, req, "MessageDoesNotExist")
}

func DeleteQueue(w http.ResponseWriter, req *http.Request) {
	// Sent response type
	w.Header().Set("Content-Type", "application/xml")

	// Retrieve FormValues required
	queueUrl := getQueueFromPath(req.FormValue("QueueUrl"), req.URL.String())
	queueName := ""
	if queueUrl == "" {
		vars := mux.Vars(req)
		queueName = vars["queueName"]
	} else {
		uriSegments := strings.Split(queueUrl, "/")
		queueName = uriSegments[len(uriSegments)-1]
	}

	log.Println("Deleting Queue:", queueName)
	SyncQueues.Lock()
	delete(SyncQueues.Queues, queueName)
	SyncQueues.Unlock()

	// Create, encode/xml and send response
	respStruct := DeleteMessageResponse{"http://queue.amazonaws.com/doc/2012-11-05/", common.ResponseMetadata{RequestId: "00000000-0000-0000-0000-000000000000"}}
	enc := xml.NewEncoder(w)
	enc.Indent("  ", "    ")
	if err := enc.Encode(respStruct); err != nil {
		log.Printf("error: %v\n", err)
	}
}

func PurgeQueue(w http.ResponseWriter, req *http.Request) {
	// Sent response type
	w.Header().Set("Content-Type", "application/xml")

	// Retrieve FormValues required
	queueUrl := getQueueFromPath(req.FormValue("QueueUrl"), req.URL.String())

	uriSegments := strings.Split(queueUrl, "/")
	queueName := uriSegments[len(uriSegments)-1]

	log.Println("Purging Queue:", queueName)

	SyncQueues.Lock()
	if _, ok := SyncQueues.Queues[queueName]; ok {
		SyncQueues.Queues[queueName].Messages = nil
		respStruct := PurgeQueueResponse{"http://queue.amazonaws.com/doc/2012-11-05/", common.ResponseMetadata{RequestId: "00000000-0000-0000-0000-000000000000"}}
		enc := xml.NewEncoder(w)
		enc.Indent("  ", "    ")
		if err := enc.Encode(respStruct); err != nil {
			log.Printf("error: %v\n", err)
			createErrorResponse(w, req, "GeneralError")
		}
	} else {
		log.Println("Purge Queue:", queueName, ", queue does not exist!!!")
		createErrorResponse(w, req, "QueueNotFound")
	}
	SyncQueues.Unlock()
}

func GetQueueUrl(w http.ResponseWriter, req *http.Request) {
	// Sent response type
	w.Header().Set("Content-Type", "application/xml")
	//
	//// Retrieve FormValues required
	queueName := req.FormValue("QueueName")
	SyncQueues.RLock()
	if queue, ok := SyncQueues.Queues[queueName]; ok {
		url := queue.URL
		log.Println("Get Queue URL:", queueName)
		// Create, encode/xml and send response
		result := GetQueueUrlResult{QueueUrl: url}
		respStruct := GetQueueUrlResponse{"http://queue.amazonaws.com/doc/2012-11-05/", result, common.ResponseMetadata{RequestId: "00000000-0000-0000-0000-000000000000"}}
		enc := xml.NewEncoder(w)
		enc.Indent("  ", "    ")
		if err := enc.Encode(respStruct); err != nil {
			log.Printf("error: %v\n", err)
		}
	} else {
		log.Println("Get Queue URL:", queueName, ", queue does not exist!!!")
		createErrorResponse(w, req, "QueueNotFound")
	}
	SyncQueues.RUnlock()
}

func GetQueueAttributes(w http.ResponseWriter, req *http.Request) {
	// Sent response type
	w.Header().Set("Content-Type", "application/xml")
	// Retrieve FormValues required
	queueUrl := getQueueFromPath(req.FormValue("QueueUrl"), req.URL.String())

	queueName := ""
	if queueUrl == "" {
		vars := mux.Vars(req)
		queueName = vars["queueName"]
	} else {
		uriSegments := strings.Split(queueUrl, "/")
		queueName = uriSegments[len(uriSegments)-1]
	}

	log.Println("Get Queue Attributes:", queueName)
	if queue, ok := SyncQueues.Queues[queueName]; ok {
		SyncQueues.RLock()
		// Create, encode/xml and send response
		attribs := make([]Attribute, 0, 0)
		attr := Attribute{Name: "VisibilityTimeout", Value: strconv.Itoa(queue.TimeoutSecs)}
		attribs = append(attribs, attr)
		attr = Attribute{Name: "DelaySeconds", Value: "0"}
		attribs = append(attribs, attr)
		attr = Attribute{Name: "ReceiveMessageWaitTimeSeconds", Value: "0"}
		attribs = append(attribs, attr)
		attr = Attribute{Name: "ApproximateNumberOfMessages", Value: strconv.Itoa(len(queue.Messages))}
		attribs = append(attribs, attr)
		attr = Attribute{Name: "ApproximateNumberOfMessagesNotVisible", Value: strconv.Itoa(numberOfHiddenMessagesInQueue(*queue))}
		attribs = append(attribs, attr)
		attr = Attribute{Name: "CreatedTimestamp", Value: "0000000000"}
		attribs = append(attribs, attr)
		attr = Attribute{Name: "LastModifiedTimestamp", Value: "0000000000"}
		attribs = append(attribs, attr)
		attr = Attribute{Name: "QueueArn", Value: queue.Arn}
		attribs = append(attribs, attr)
		SyncQueues.RUnlock()

		result := GetQueueAttributesResult{Attrs: attribs}
		respStruct := GetQueueAttributesResponse{"http://queue.amazonaws.com/doc/2012-11-05/", result, common.ResponseMetadata{RequestId: "00000000-0000-0000-0000-000000000000"}}
		enc := xml.NewEncoder(w)
		enc.Indent("  ", "    ")
		if err := enc.Encode(respStruct); err != nil {
			log.Printf("error: %v\n", err)
		}
	} else {
		log.Println("Get Queue URL:", queueName, ", queue does not exist!!!")
		createErrorResponse(w, req, "QueueNotFound")
	}
}

func SetQueueAttributes(w http.ResponseWriter, req *http.Request) {
	log.Println("setQueueAttributes was called but it's not implemented")
	respStruct := SetQueueAttributesResponse{"http://queue.amazonaws.com/doc/2012-11-05/", common.ResponseMetadata{RequestId: "00000000-0000-0000-0000-000000000000"}}
	enc := xml.NewEncoder(w)
	enc.Indent("  ", "    ")
	if err := enc.Encode(respStruct); err != nil {
		log.Printf("error: %v\n", err)
		createErrorResponse(w, req, "GeneralError")
	}
}

func getQueueFromPath(formVal string, theUrl string) string {
	if formVal != "" {
		return formVal
	}
	u, err := url.Parse(theUrl)
	if err != nil {
		return ""
	}
	return u.Path
}

func createErrorResponse(w http.ResponseWriter, req *http.Request, err string) {
	er := SqsErrors[err]
	respStruct := common.ErrorResponse{common.ErrorResult{Type: er.Type, Code: er.Code, Message: er.Message, RequestId: "00000000-0000-0000-0000-000000000000"}}

	w.WriteHeader(er.HttpError)
	enc := xml.NewEncoder(w)
	enc.Indent("  ", "    ")
	if err := enc.Encode(respStruct); err != nil {
		log.Printf("error: %v\n", err)
	}
}
