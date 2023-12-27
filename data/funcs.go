package data

import "encoding/json"

func FromBytes(bytes []byte) (Event, error) {
	var frm Event
	err := json.Unmarshal(bytes, &frm)
	return frm, err
}
