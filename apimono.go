package apimono

import (
	"context"
	"encoding/json"
	"io"
)

const (
	defaultJsonLevel = 0
	defaultJsonToken = json.Delim('[')
)

var (
	noopNext = func(_ int) (string, bool) {
		return "", false
	}
	noopProvider = func(c context.Context, s string) (io.ReadCloser, error) {
		return nil, nil
	}
)

func Build(ctx context.Context, opts ...RowsOption) (*Rows, error) {
	rows := &Rows{
		ctx:      ctx,
		level:    defaultJsonLevel,
		token:    defaultJsonToken,
		nextpage: noopNext,
		provider: noopProvider,
	}
	for _, opt := range opts {
		opt(rows)
	}
	return rows, nil
}
