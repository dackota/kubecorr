// kubecorr merges Kubernetes pod logs and events into one timeline.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/dackota/kubecorr/cmd"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if err := cmd.New().ExecuteContext(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
