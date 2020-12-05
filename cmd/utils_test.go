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

import "testing"

func TestNewSHA256(t *testing.T) {
	t.Parallel()

	got := NewSHA256([]byte("dsdd"))
	if got != "701df70cc797a5d18f69fbf8fa538b15c5adcc06e51de80b446d465696d6c3b5" {
		t.Error("SHA256 is not correct")
	}
}
