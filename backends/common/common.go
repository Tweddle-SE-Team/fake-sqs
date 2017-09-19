package common

import (
	"crypto/md5"
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"sort"

	log "github.com/sirupsen/logrus"
)

var LogMessages bool
var LogFile string

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

func ExtractMessageAttributes(req *http.Request, service string) map[string]MessageAttribute {
	attributes := make(map[string]MessageAttribute)

	for i := 1; true; i++ {
		pattern := fmt.Sprintf("MessageAttribute.%d.Name", i)
		if service == "sns" {
			pattern = fmt.Sprintf("MessageAttributes.entry.%d.Name", i)
		} else {
			log.Warnf("Handler for %s service is undefined!\n", service)
			break
		}
		name := req.FormValue(pattern)
		if name == "" {
			break
		}

		pattern = fmt.Sprintf("MessageAttribute.%d.Value.DataType", i)
		if service == "sns" {
			pattern = fmt.Sprintf("MessageAttributes.entry.%d.Value.DataType", i)
		} else {
			log.Warnf("Handler for %s service is undefined!\n", service)
			break
		}

		dataType := req.FormValue(pattern)
		if dataType == "" {
			log.Warnf("DataType of MessageAttribute %s is missing, MD5 checksum will most probably be wrong!\n", name)
			continue
		}

		// StringListValue and BinaryListValue is currently not implemented
		for _, valueKey := range [...]string{"StringValue", "BinaryValue"} {

			pattern = fmt.Sprintf("MessageAttribute.%d.Value.%s", i, valueKey)
			if service == "sns" {
				pattern = fmt.Sprintf("MessageAttributes.entry.%d.Value.%s", i, valueKey)
			} else {
				log.Warnf("Handler for %s service is undefined!\n", service)
				break
			}

			value := req.FormValue(pattern)
			if value != "" {
				messageAttributeValue := MessageAttributeValue{DataType: dataType}
				if valueKey == "StringValue" {
					messageAttributeValue.StringValue = value
				} else if valueKey == "BinaryValue" {
					messageAttributeValue.BinaryValue = value
				}
				attributes[name] = MessageAttribute{Name: name, Value: messageAttributeValue}
			}
		}

		if _, ok := attributes[name]; !ok {
			log.Warnf("StringValue or BinaryValue of MessageAttribute %s is missing, MD5 checksum will most probably be wrong!\n", name)
		}
	}

	return attributes
}

func HashAttributes(attributes map[string]MessageAttribute) string {
	hasher := md5.New()

	keys := sortedKeys(attributes)
	for _, key := range keys {
		attributeValue := attributes[key]
		addStringToHash(hasher, key)
		addStringToHash(hasher, attributeValue.Value.DataType)
		if attributeValue.Value.StringValue != "" {
			hasher.Write([]byte{1})
			addStringToHash(hasher, attributeValue.Value.StringValue)
		} else if attributeValue.Value.BinaryValue != "" {
			hasher.Write([]byte{2})
			bytes, _ := base64.StdEncoding.DecodeString(attributeValue.Value.BinaryValue)
			addBytesToHash(hasher, bytes)
		}
	}

	return hex.EncodeToString(hasher.Sum(nil))
}

func sortedKeys(attributes map[string]MessageAttribute) []string {
	var keys []string
	for key, _ := range attributes {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func addStringToHash(hasher hash.Hash, str string) {
	bytes := []byte(str)
	addBytesToHash(hasher, bytes)
}

func addBytesToHash(hasher hash.Hash, arr []byte) {
	bs := make([]byte, 4)
	binary.BigEndian.PutUint32(bs, uint32(len(arr)))
	hasher.Write(bs)
	hasher.Write(arr)
}
