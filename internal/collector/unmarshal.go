package collector

import (
	"encoding/json"
)

// When unmarshalling JSON from iDRAC, the "xstring" type defined here can be
// one of the following:
// - nil
// - string
// - [{"Member": "VALUE"}]
type xstring string

func (w *xstring) UnmarshalJSON(data []byte) (err error) {
	var x interface{}

	err = json.Unmarshal(data, &x)
	if err != nil {
		return err
	}

	if x == nil {
		*w = xstring("")
		return nil
	}

	s, ok := x.(string)
	if ok {
		*w = xstring(s)
		return nil
	}

	list := x.([]interface{})
	dict := list[0].(map[string]interface{})
	s = dict["Member"].(string)

	*w = xstring(s)
	return nil
}
