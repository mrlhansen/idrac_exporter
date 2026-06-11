package collector

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// When unmarshalling JSON, the "xstring" type can be one of the following
// - nil
// - string
// - float64
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

	// case: nil
	if x == nil {
		return nil
	}

	// case: string
	if v, ok := x.(string); ok {
		s = v
		return nil
	}

	// case: int
	if v, ok := x.(int); ok {
		s = fmt.Sprintf("%v", v)
		return nil
	}

	// case: float64
	if v, ok := x.(float64); ok {
		s = fmt.Sprintf("%v", v)
		return nil
	}

	// strange iDRAC case
	list, ok := x.([]any)
	if !ok {
		return nil
	}

	dict, ok := list[0].(map[string]any)
	if !ok {
		return nil
	}

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

// asFloat64 coerces a value decoded into an "any" field (such as ReadingVolts,
// which some BMCs serialize as a JSON string) into a float64.
func asFloat64(v any) (float64, bool) {
	switch x := v.(type) {
	case float64:
		return x, true
	case string:
		f, err := strconv.ParseFloat(strings.TrimSpace(x), 64)
		if err != nil {
			return 0, false
		}
		return f, true
	}
	return 0, false
}
