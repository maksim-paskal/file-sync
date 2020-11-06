package main

import (
	"context"
)

func main() {
	ctx := context.Background()

	config := newConfig()

	newWeb(config)

	<-ctx.Done()
}
