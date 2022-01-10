package client

import (
	"encoding/json"
	"time"
)

type CTime time.Time

func (ct *CTime) UnmarshalJSON(bytes []byte) error {
	var raw string
	err := json.Unmarshal(bytes, &raw)
	if err != nil {
		return err
	}
	v, err := time.Parse("Jan _2 15:04:05 2006 MST", raw)
	if err != nil {
		return err
	}
	*ct = CTime(v)
	return err
}

func (ct *CTime) MarshalJSON() ([]byte, error) {
	return json.Marshal(ct.Time())
}

func (ct CTime) Time() time.Time {
	return time.Time(ct)
}
