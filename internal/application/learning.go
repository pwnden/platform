package application

import (
	"fmt"

	"github.com/pwnden/platform/internal/challenge"
)

type Learning struct {
	Requires []challenge.Concept
	Teaches  []challenge.Concept
}

func learning(c *challenge.Loaded, concepts map[string]challenge.Concept) (*Learning, error) {
	if c.Schema < 7 {
		return nil, nil
	}
	if c.Learning == nil {
		return nil, fmt.Errorf("missing learning declaration")
	}
	result := &Learning{Requires: []challenge.Concept{}, Teaches: []challenge.Concept{}}
	seen := map[string]bool{}
	queue := append([]string{}, c.Learning.Requires...)
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		if seen[id] {
			continue
		}
		item, exists := concepts[id]
		if !exists {
			return nil, fmt.Errorf("unknown learning concept %q", id)
		}
		seen[id] = true
		result.Requires = append(result.Requires, item)
		queue = append(queue, item.Requires...)
	}
	for _, id := range c.Learning.Teaches {
		item, exists := concepts[id]
		if !exists {
			return nil, fmt.Errorf("unknown learning concept %q", id)
		}
		result.Teaches = append(result.Teaches, item)
	}
	return result, nil
}
