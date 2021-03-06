/*
Copyright paskal.maksim@gmail.com
Licensed under the Apache License, Version 2.0 (the "License")
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
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
	t.Parallel()

	t.Logf("config.sourceDir=%s", *appConfig.sourceDir)

	tests := make([]TestAPIItem, 6)

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
		value: "copy:tests/test.txt:tests/test2.txt",
		message: Message{
			Type:              "copy",
			FileName:          "tests/test.txt",
			NewFileName:       "tests/test2.txt",
			Force:             false,
			FileContentBase64: "",
			SHA256:            "",
		},
	}

	tests[3] = TestAPIItem{
		value: "move:tests/test2.txt:tests/test/test/test/test3.txt",
		message: Message{
			Type:              "move",
			FileName:          "tests/test2.txt",
			NewFileName:       "tests/test/test/test/test3.txt",
			Force:             false,
			FileContentBase64: "",
			SHA256:            "",
		},
	}

	tests[4] = TestAPIItem{
		value: "delete:tests/test/test/test/test3.txt",
		message: Message{
			Type:     "delete",
			FileName: "tests/test/test/test/test3.txt",
		},
	}

	tests[5] = TestAPIItem{
		value: "delete:tests/test.txt",
		message: Message{
			Type:     "delete",
			FileName: "tests/test.txt",
		},
	}

	api, err := newAPI()
	if err != nil {
		t.Fatal(err)
	}

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

		err = api.processMessage(test.message)

		if err != nil {
			t.Error(err)

			return
		}
	}
}
