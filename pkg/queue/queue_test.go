// +build integration

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
package queue_test

import (
	"context"
	"github.com/google/uuid"
	"io/ioutil"
	"strconv"
	"testing"

	"github.com/maksim-paskal/file-sync/pkg/api"
	"github.com/maksim-paskal/file-sync/pkg/config"
	"github.com/maksim-paskal/file-sync/pkg/queue"
	log "github.com/sirupsen/logrus"
)

var ctx = context.Background()

func init() {
	if err := config.Load(); err != nil {
		panic(err)
	}

	if err := queue.Init(); err != nil {
		panic(err)
	}
}

func TestAdd(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	message := api.Message{
		Type: uuid.NewString(),
	}

	_, err := queue.Info()
	if err != nil {
		t.Fatal(err)
	}

	queueSize := 100
	queueFound := 0

	queue.OnNewValue = func(m api.Message) {
		log.Info(m.ID)

		if m.Type != message.Type {
			log.Fatal("type mismatch")
		}

		queueFound++

		log.Info(queueFound)

		_ = ioutil.WriteFile("/tmp/eee", []byte(strconv.Itoa(queueFound)), 0777)

		if queueFound == queueSize {
			cancel()
		}
	}

	for i := 0; i < queueSize; i++ {
		id, err := queue.Add(message)
		if err != nil {
			t.Fatal(err)
		}

		t.Log(id)
	}

	<-ctx.Done()
}
