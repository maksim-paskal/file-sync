package main

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestGetMessageFromValue(t *testing.T) {
	config := newConfig()

	sourceDir := "../examples"
	destinationDir := "../data-test"

	config.sourceDir = &sourceDir
	config.destinationDir = &destinationDir

	t.Logf("config.sourceDir=%s", *config.sourceDir)

	tests := make(map[string]Message)

	tests["put:tests/test.txt"] = Message{
		Type:              "put",
		FileName:          "tests/test.txt",
		FileContentBase64: "ZHNkZA==",
		SHA256:            "701df70cc797a5d18f69fbf8fa538b15c5adcc06e51de80b446d465696d6c3b5",
	}

	api := newAPI(config)

	for key, message := range tests {
		result, err := api.getMessageFromValue(key)
		if err != nil {
			t.Error(err)

			return
		}

		js, _ := json.Marshal(result)
		t.Log(string(js))

		if !reflect.DeepEqual(result, message) {
			t.Error("messages not correct")

			return
		}

		err = api.makePUT(message)
		if err != nil {
			t.Error(err)

			return
		}
	}
}
