package helper

import (
	"encoding/json"
	"fmt"
)

func ToJSON(v any) (string, error) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal JSON: %w", err)
	}

	return string(data), nil
}
