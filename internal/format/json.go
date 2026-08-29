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

// JSONLine writes one item as a single JSON line (for streaming).
func JSONLine(w io.Writer, it timeline.Item) error {
	if err := json.NewEncoder(w).Encode(it); err != nil {
		return fmt.Errorf("encode json line: %w", err)
	}
	return nil
}
