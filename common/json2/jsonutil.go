package json2

import "encoding/json"

func ToJSON(v interface{}) []byte {
	b, _ := json.Marshal(v)
	return b
}

func ToJSONString(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func ToPrettyJSON(v interface{}) string {
	b, _ := json.MarshalIndent(v, "", "  ")
	return string(b)
}
