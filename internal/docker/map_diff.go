package docker

import (
	"fmt"
	"sort"
)

func getDiff[T fmt.Stringer](old, new map[string]T) []string {
	changes := make([]string, 0, len(old)+len(new))
	for k, v := range old {
		if n, ok := new[k]; ok {
			if n.String() != v.String() {
				// Changed
				changes = append(changes, fmt.Sprintf("# %s -> %s", v, n))
			}
			continue
		}
		changes = append(changes, fmt.Sprintf("- %s", v))
	}
	for k, v := range new {
		if _, ok := old[k]; !ok {
			changes = append(changes, fmt.Sprintf("+ %s", v))
		}
	}
	sort.Strings(changes)
	return changes
}
