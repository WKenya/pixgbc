package export

import "encoding/json"

func JSONBytes(v any) ([]byte, error) {
	return json.MarshalIndent(v, "", "  ")
}
