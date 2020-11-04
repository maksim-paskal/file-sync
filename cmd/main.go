package main

import (
	"context"
	"fmt"
)

func main() {
	config := newConfig()

	ctx := context.Background()

	queue := newQueue(config, "test")

	queue.onNewValue = func(value string) {
		fmt.Printf("new=%s\n", value)
	}

	newWeb(config, queue)

	<-ctx.Done()
}
