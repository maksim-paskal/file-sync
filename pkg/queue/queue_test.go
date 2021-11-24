//go:build integration
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
	"strconv"
	"testing"

	"github.com/google/uuid"
	"github.com/maksim-paskal/file-sync/pkg/api"
	"github.com/maksim-paskal/file-sync/pkg/config"
	"github.com/maksim-paskal/file-sync/pkg/queue"
)

var ctx = context.Background()

func init() { //nolint:gochecknoinits
	if err := config.Load(); err != nil {
		panic(err)
	}

	if err := queue.Init(); err != nil {
		panic(err)
	}

	if err := queue.Flush(); err != nil {
		panic(err)
	}
}

func TestAdd(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	messageType := uuid.NewString()

	_, err := queue.Info()
	if err != nil {
		t.Fatal(err)
	}

	const queueSize = 1000

	queueFound := 0
	lastID := -1

	queue.OnNewValue = func(m api.Message) {
		t.Logf("OnNewValue=%s", m.ID)

		currentID, err := strconv.Atoi(m.ID)
		if err != nil {
			t.Fatalf("%v,%s", err, m.ID)
			cancel()
			queue.GracefullShutdown()
		}

		if lastID >= 0 && (currentID-lastID) != 1 {
			t.Errorf("wrong order currentID=%d, lastID=%d", currentID, lastID)
			cancel()
			queue.GracefullShutdown()
		}

		lastID = currentID

		if m.Type != messageType {
			t.Errorf("type mismatch m.Type=%s,messageType=%s", m.Type, messageType)
			cancel()
			queue.GracefullShutdown()
		}

		queueFound++

		if queueFound == queueSize {
			cancel()
		}
	}

	for i := 0; i < queueSize; i++ {
		message := api.Message{
			ID:   strconv.Itoa(i),
			Type: messageType,
		}

		_, err := queue.Add(message)
		if err != nil {
			t.Fatal(err)
		}
	}

	<-ctx.Done()
}
