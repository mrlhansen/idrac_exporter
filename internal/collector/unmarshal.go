package collector

import (
	"encoding/json"
	"fmt"
)

// When unmarshalling JSON, the "xstring" type can be one of the following
// - nil
// - string
// - integer
// - [{"Member": "VALUE"}]
type xstring string

func (w *xstring) UnmarshalJSON(data []byte) error {
	var x any
	var s string

	defer func() {
		*w = xstring(s)
	}()

	err := json.Unmarshal(data, &x)
	if err != nil {
		return err
	}

	if x == nil {
		return nil
	}

	if v, ok := x.(string); ok {
		s = v
		return nil
	}

	if v, ok := x.(int); ok {
		s = fmt.Sprintf("%d", v)
		return nil
	}

	list := x.([]any)
	dict := list[0].(map[string]any)

	if v, ok := dict["Member"].(string); ok {
		s = v
		return nil
	}

	return nil
}

func (w *xstring) String() string {
	return string(*w)
}

func asPtr[T any](v T) *T {
	return &v
}
