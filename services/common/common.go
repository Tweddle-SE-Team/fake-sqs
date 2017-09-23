package common

import (
	"crypto/md5"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"hash"
	"io"
	"io/ioutil"
	"os"
	"sort"

	log "github.com/sirupsen/logrus"
)

var LogMessages bool
var LogFile string

type (
	Protocol         string
	MessageStructure string
	ErrorHandler     map[string]ErrorType
)

const (
	ProtocolSQS          Protocol         = "sqs"
	ProtocolDefault      Protocol         = "default"
	MessageStructureJson MessageStructure = "json"
)

func NewUUID() (string, error) {
	uuid := make([]byte, 16)
	n, err := io.ReadFull(rand.Reader, uuid)
	if n != len(uuid) || err != nil {
		return "", err
	}
	// variant bits; see section 4.1.1
	uuid[8] = uuid[8]&^0xc0 | 0x80
	// version 4 (pseudo-random); see section 4.1.3
	uuid[6] = uuid[6]&^0xf0 | 0x40
	return fmt.Sprintf("%x-%x-%x-%x-%x", uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:]), nil
}

func GetMD5Hash(text string) string {
	hasher := md5.New()
	hasher.Write([]byte(text))
	return hex.EncodeToString(hasher.Sum(nil))
}

func LogMessage(msg string) {
	if _, err := os.Stat(LogFile); os.IsNotExist(err) {
		_, err := os.Create("/tmp/dat2")
		if err != nil {
			log.Println("could not create log file:", LogFile)
			return
		}
	}
	if LogMessages == true {
		ioutil.WriteFile(LogFile, []byte(msg), 0644)
	}
}

func ExtractMessageBodyFromJson(msg string, protocol string) (*string, error) {
	var msgWithProtocols map[string]string
	if err := json.Unmarshal([]byte(msg), &msgWithProtocols); err != nil {
		return nil, err
	}
	defaultMsg, ok := msgWithProtocols[string(ProtocolDefault)]
	if !ok {
		return nil, errors.New("Invalid parameter: Message Structure - No default entry in JSON message body")
	}
	if m, ok := msgWithProtocols[protocol]; ok {
		return &m, nil
	}
	return &defaultMsg, nil
}

func AddStringToHash(hasher hash.Hash, str string) {
	bytes := []byte(str)
	AddBytesToHash(hasher, bytes)
}

func AddBytesToHash(hasher hash.Hash, arr []byte) {
	bs := make([]byte, 4)
	binary.BigEndian.PutUint32(bs, uint32(len(arr)))
	hasher.Write(bs)
	hasher.Write(arr)
}

func SortKeys(a interface{}) []string {
	attributes := make(map[string]interface{})
	var keys []string
	for key, _ := range attributes {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

/*** Common ***/
type ResponseMetadata struct {
	RequestId string `xml:"RequestId"`
}

/*** Error Responses ***/
type ErrorResult struct {
	Type      string `xml:"Type,omitempty"`
	Code      string `xml:"Code,omitempty"`
	Message   string `xml:"Message,omitempty"`
	RequestId string `xml:"RequestId,omitempty"`
}

type ErrorType struct {
	HttpError int
	Type      string
	Code      string
	Message   string
}

type ErrorResponse struct {
	Result ErrorResult `xml:"Error"`
}
