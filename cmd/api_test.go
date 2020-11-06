package main

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestGetMessageFromValue(t *testing.T) {
	sourceDir := "../examples"
	destinationDir := "../data-test"

	appConfig.sourceDir = &sourceDir
	appConfig.destinationDir = &destinationDir

	t.Logf("config.sourceDir=%s", *appConfig.sourceDir)

	tests := make(map[string]Message)

	tests["put:tests/test.txt"] = Message{
		Type:              "put",
		FileName:          "tests/test.txt",
		FileContentBase64: "ZHNkZA==",
		SHA256:            "701df70cc797a5d18f69fbf8fa538b15c5adcc06e51de80b446d465696d6c3b5",
	}

	tests["patch:tests/test.txt"] = Message{
		Type:              "patch",
		FileName:          "tests/test.txt",
		FileContentBase64: "ZHNkZA==",
		SHA256:            "701df70cc797a5d18f69fbf8fa538b15c5adcc06e51de80b446d465696d6c3b5",
	}

	tests["delete:tests/test.txt"] = Message{
		Type:     "delete",
		FileName: "tests/test.txt",
	}

	api := newAPI()

	for key, message := range tests {
		t.Log(key)

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

		switch message.Type {
		case MessageTypePut:
			err = api.makeSave(message)
		case MessageTypePatch:
			err = api.makeSave(message)
		case MessageTypeDelete:
			err = api.makeDelete(message)
		}

		if err != nil {
			t.Error(err)

			return
		}
	}
}
