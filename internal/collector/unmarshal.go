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

func (w *xstring) UnmarshalJSON(data []byte) error {
	var x any

	err := json.Unmarshal(data, &x)
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

	list := x.([]any)
	dict := list[0].(map[string]any)
	s, ok = dict["Member"].(string)
	if ok {
		*w = xstring(s)
		return nil
	}

	*w = xstring("")
	return nil
}

func (w *xstring) String() string {
	return string(*w)
}
