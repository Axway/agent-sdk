package filter

import (
	"reflect"
	"strings"
)

const (
	filterTypeAttr = "attr"
	filterTypeTag  = "tag"
)

// Data - Interface representing the data for filter evaluation
type Data interface {
	GetValue(filterType string, filterKey string) (string, bool)
	GetKeys(filterType string) []string
	GetValues(filterType string) []string
}

// AgentFilterData - Represents the data for filter evaluation
type AgentFilterData struct {
	tags map[string]string
	attr map[string]string
}

// NewFilterData - Transforms the data to flat map which is used for filter evaluation
func NewFilterData(tags interface{}, attr interface{}) Data {
	vTags := reflect.ValueOf(tags)
	tagsMap := make(map[string]string)
	// Todo address other types
	if vTags.Kind() == reflect.Map {
		for _, key := range vTags.MapKeys() {
			tagKey := strings.ReplaceAll(key.String(), Dash, DashPlaceHolder)
			value := vTags.MapIndex(key)
			vInterface := reflect.ValueOf(value.Interface())
			if vInterface.Kind() == reflect.Ptr {
				vInterface = vInterface.Elem()
			}
			if vInterface.Kind() == reflect.String {
				keyValue := vInterface.String()
				tagsMap[tagKey] = keyValue
			}
			if vInterface.Kind() == reflect.Slice {
				tagsMap[tagKey] = parseStringSliceFilterData(vInterface)
			}
		}
	}

	return &AgentFilterData{
		tags: tagsMap,
	}
}

func parseStringSliceFilterData(v reflect.Value) string {
	var val = ""
	for i := 0; i < v.Len(); i++ {
		vItem := v.Index(i)
		if vItem.Kind() == reflect.String {
			if len(val) > 0 {
				val += ","
			}
			val += vItem.String()
		}
	}
	return val
}

// GetKeys - Returns all the map keys based on the filter data type
func (ad *AgentFilterData) GetKeys(ftype string) []string {
	keys := make([]string, 0)
	var m map[string]string
	if ftype == filterTypeAttr {
		m = ad.attr
	} else {
		m = ad.tags
	}

	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// GetValues - Returns all the map values based on the filter data type
func (ad *AgentFilterData) GetValues(ftype string) []string {
	values := make([]string, 0)
	var m map[string]string
	if ftype == filterTypeAttr {
		m = ad.attr
	} else {
		m = ad.tags
	}
	for _, v := range m {
		values = append(values, v)
	}
	return values
}

// GetValue - Returns the value for map entry based on the filter data type
func (ad *AgentFilterData) GetValue(ftype, fName string) (val string, ok bool) {
	var m map[string]string
	if ftype == filterTypeAttr {
		m = ad.attr
	} else {
		m = ad.tags
	}
	if m != nil {
		val, ok = m[fName]
	}
	return
}
