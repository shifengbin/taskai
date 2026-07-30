package lifecycle

import (
	"encoding/json"

	"taskai/internal/realtime"
)

func BuildCommandInput(resource realtime.TaskResource, baseURL string) ([]byte, error) {
	input := struct {
		realtime.TaskResource
		BaseURL string `json:"baseURL,omitempty"`
	}{
		TaskResource: resource,
		BaseURL:      baseURL,
	}
	return json.Marshal(input)
}
