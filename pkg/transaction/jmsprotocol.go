package transaction

import (
	"reflect"
	"strconv"
	"strings"
)

// JMSProtocol - Represents the details in a transaction event for the JMS protocol
type JMSProtocol struct {
	Type             string `json:"type,omitempty"`
	AuthSubjectID    string `json:"authSubjectId,omitempty"`
	JMSMessageID     string `json:"jmsMessageID,omitempty"`
	JMSCorrelationID string `json:"jmsCorrelationID,omitempty"`
	JMSDestination   string `json:"jmsDestination,omitempty"`
	JMSProviderURL   string `json:"jmsProviderURL,omitempty"`
	JMSDeliveryMode  int    `json:"jmsDeliveryMode,omitempty"`
	JMSPriority      int    `json:"jmsPriority,omitempty"`
	JMSReplyTo       string `json:"jmsReplyTo,omitempty"`
	JMSRedelivered   int    `json:"jmsRedelivered,omitempty"`
	JMSTimestamp     int    `json:"jmsTimestamp,omitempty"`
	JMSExpiration    int    `json:"jmsExpiration,omitempty"`
	JMSType          string `json:"jmsType,omitempty"`
	JMSStatus        string `json:"jmsStatus,omitempty"`
	JMSStatusText    string `json:"jmsStatusText,omitempty"`
}

// ToMapStringString - convert the JMSProtocol to a map[string]string
func (j *JMSProtocol) ToMapStringString() (map[string]string, error) {
	jmsPropertyStringMap := make(map[string]string)

	value := reflect.ValueOf(j).Elem()
	for i := 0; i < value.NumField(); i++ {
		field := value.Field(i)
		jsonField := value.Type().Field(i).Tag.Get("json")
		jsonKey := strings.Split(jsonField, ",")[0]

		switch field.Kind() {
		case reflect.String:
			jmsPropertyStringMap[jsonKey] = field.String()
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			jmsPropertyStringMap[jsonKey] = strconv.FormatInt(field.Int(), 10)
		}
	}
	return jmsPropertyStringMap, nil
}

// FromMapStringString - convert the map[string]string to a JMSProtocol
func (j *JMSProtocol) FromMapStringString(propertyMap map[string]string) error {
	ignoreFields := map[string]bool{
		"Type":         true,
		"JMSStatus":    true,
		"JMSTimestamp": true,
	}

	value := reflect.ValueOf(j).Elem()
	for i := 0; i < value.NumField(); i++ {
		field := value.Field(i)
		fieldName := value.Type().Field(i).Name
		jsonField := value.Type().Field(i).Tag.Get("json")

		jsonKey := strings.Split(jsonField, ",")[0]
		propertyMapValue, found := propertyMap[jsonKey]

		if _, ignore := ignoreFields[fieldName]; ignore {
			// Skip and ignore fields
			continue
		}

		switch field.Kind() {
		case reflect.String:
			// update the fields value
			value.FieldByName(fieldName).SetString("")
			if found {
				value.FieldByName(fieldName).SetString(propertyMapValue)
			}
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			// Default the value to 0, set to new value if int
			value.FieldByName(fieldName).SetInt(0)
			if intValue, err := strconv.ParseInt(propertyMapValue, 10, 64); found && err == nil {
				value.FieldByName(fieldName).SetInt(intValue)
			}
		}
	}
	return nil
}
