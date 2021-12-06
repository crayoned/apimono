package apimono

import (
	"context"
	"encoding/json"
	"io"
)

type RowsOption func(*Rows)

func WithToken(token json.Token, level int) RowsOption {
	return func(r *Rows) {
		r.token = token
		r.level = level
	}
}

func WithNext(nextpage func(int) (string, bool)) RowsOption {
	return func(r *Rows) {
		r.nextpage = nextpage
	}
}

func WithProvider(provider func(context.Context, string) (io.ReadCloser, error)) RowsOption {
	return func(r *Rows) {
		r.provider = provider
	}
}
