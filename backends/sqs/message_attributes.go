package sqs

import (
	"crypto/md5"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"hash"
	"net/http"
	"sort"

	log "github.com/sirupsen/logrus"
)

func extractMessageAttributes(req *http.Request) map[string]MessageAttribute {
	attributes := make(map[string]MessageAttribute)

	for i := 1; true; i++ {
		name := req.FormValue(fmt.Sprintf("MessageAttribute.%d.Name", i))
		if name == "" {
			break
		}

		dataType := req.FormValue(fmt.Sprintf("MessageAttribute.%d.Value.DataType", i))
		if dataType == "" {
			log.Warnf("DataType of MessageAttribute %s is missing, MD5 checksum will most probably be wrong!\n", name)
			continue
		}

		// StringListValue and BinaryListValue is currently not implemented
		for _, valueKey := range [...]string{"StringValue", "BinaryValue"} {
			value := req.FormValue(fmt.Sprintf("MessageAttribute.%d.Value.%s", i, valueKey))
			if value != "" {
				messageAttributeValue := MessageAttributeValue{StringValue: valueKey, DataType: dataType}
				attributes[name] = MessageAttribute{Name: name, Value: messageAttributeValue}
			}
		}

		if _, ok := attributes[name]; !ok {
			log.Warnf("StringValue or BinaryValue of MessageAttribute %s is missing, MD5 checksum will most probably be wrong!\n", name)
		}
	}

	return attributes
}

func hashAttributes(attributes map[string]MessageAttribute) string {
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
