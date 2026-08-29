package timeline

import "sort"

// Merge returns a new slice with every item from both inputs, sorted by time.
// Inputs are not changed. Items with equal time keep their input order.
func Merge(logs, events []Item) []Item {
	out := make([]Item, 0, len(logs)+len(events))
	out = append(out, logs...)
	out = append(out, events...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].Time.Before(out[j].Time) })
	return out
}
