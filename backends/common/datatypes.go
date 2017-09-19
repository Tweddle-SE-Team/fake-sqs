package common

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

type ErrorResponse struct {
	Result ErrorResult `xml:"Error"`
}

/*** Get Message Attributes ***/
type MessageAttribute struct {
	Name  string                `xml:"Name,omitempty"`
	Value MessageAttributeValue `xml:"Value,omitempty"`
}

type MessageAttributeValue struct {
	StringValue     string `xml:"StringValue,omitempty" json:"StringValue,omitempty"`
	BinaryValue     string `xml:"BinaryValue,omitempty" json:"BinaryValue,omitempty"`
	BinaryListValue string `xml:"BinaryListValue,omitempty" json:"BinaryListValue,omitempty"`
	StringListValue string `xml:"StringListValue,omitempty" json:"StringListValue,omitempty"`
	DataType        string `xml:"DataType,omitempty"`
}

type GetMessageAttributesResult struct {
	MessageAttrs map[string]MessageAttribute `xml:"MessageAttribute,omitempty"`
}
