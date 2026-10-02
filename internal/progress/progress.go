// Package progress defines durable player state independently of its storage.
package progress

import (
	"context"
	"time"
)

type Completion struct {
	Slug     string
	Answer   string
	SolvedAt time.Time
}

type Store interface {
	List(context.Context) (map[string]Completion, error)
	Save(context.Context, Completion) error
}
