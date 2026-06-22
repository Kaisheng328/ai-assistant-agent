package ollama

import (
	"goyave.dev/goyave/v4"
)

func Models(response *goyave.Response, request *goyave.Request) {
	res := []map[string]interface{}{
		{
			"name":    "meta/llama-3.3-70b-instruct",
			"model":   "meta/llama-3.3-70b-instruct",
			"details": map[string]interface{}{"family": "llama", "parameter_size": "70B"},
		},
		{
			"name":    "meta/llama-3.2-3b-instruct",
			"model":   "meta/llama-3.2-3b-instruct",
			"details": map[string]interface{}{"family": "llama", "parameter_size": "3B"},
		},
		{
			"name":    "google/gemma-3-12b-it",
			"model":   "google/gemma-3-12b-it",
			"details": map[string]interface{}{"family": "gemma", "parameter_size": "12B"},
		},
		{
			"name":    "nvidia/llama-3.3-nemotron-super-49b-v1.5",
			"model":   "nvidia/llama-3.3-nemotron-super-49b-v1.5",
			"details": map[string]interface{}{"family": "llama", "parameter_size": "49B"},
		},
	}
	response.JSON(200, map[string]interface{}{"models": res})
}
