package format

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/dackota/kubecorr/internal/timeline"
)

// JSON writes the items as one JSON array. nil becomes [].
func JSON(w io.Writer, items []timeline.Item) error {
	if items == nil {
		items = []timeline.Item{}
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(items); err != nil {
		return fmt.Errorf("encode json: %w", err)
	}
	return nil
}
