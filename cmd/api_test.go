package main

import (
	"encoding/json"
	"reflect"
	"testing"
)

type TestAPIItem struct {
	value   string
	message Message
}

func TestGetMessageFromValue(t *testing.T) {
	sourceDir := "../examples"
	destinationDir := "../data-test"

	appConfig.sourceDir = &sourceDir
	appConfig.destinationDir = &destinationDir

	t.Logf("config.sourceDir=%s", *appConfig.sourceDir)

	tests := make([]TestAPIItem, 3)

	tests[0] = TestAPIItem{
		value: "put:tests/test.txt",
		message: Message{
			Type:              "put",
			FileName:          "tests/test.txt",
			FileContentBase64: "ZHNkZA==",
			SHA256:            "701df70cc797a5d18f69fbf8fa538b15c5adcc06e51de80b446d465696d6c3b5",
		},
	}

	tests[1] = TestAPIItem{
		value: "patch:tests/test.txt",
		message: Message{
			Type:              "patch",
			FileName:          "tests/test.txt",
			FileContentBase64: "ZHNkZA==",
			SHA256:            "701df70cc797a5d18f69fbf8fa538b15c5adcc06e51de80b446d465696d6c3b5",
		},
	}

	tests[2] = TestAPIItem{
		value: "delete:tests/test.txt",
		message: Message{
			Type:     "delete",
			FileName: "tests/test.txt",
		},
	}

	api := newAPI()

	for _, test := range tests {
		t.Log(test.value)

		result, err := api.getMessageFromValue(test.value)
		if err != nil {
			t.Error(err)

			return
		}

		js, _ := json.Marshal(result)
		t.Log(string(js))

		if !reflect.DeepEqual(result, test.message) {
			t.Error("messages not correct")

			return
		}

		switch test.message.Type {
		case MessageTypePut:
			err = api.makeSave(test.message)
		case MessageTypePatch:
			err = api.makeSave(test.message)
		case MessageTypeDelete:
			err = api.makeDelete(test.message)
		}

		if err != nil {
			t.Error(err)

			return
		}
	}
}
