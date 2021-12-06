package apimono

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
)

const (
	tokenArrStart  = json.Delim('[')
	tokenArrFinish = json.Delim(']')
	tokenObjStart  = json.Delim('{')
	tokenObjFinish = json.Delim('}')
)

type Rows struct {
	err error
	ctx context.Context

	dec  *json.Decoder
	body io.ReadCloser

	level int
	token json.Token

	count    int
	nextpage func(int) (string, bool)
	provider func(context.Context, string) (io.ReadCloser, error)
}

func (r *Rows) Err() error {
	return r.err
}

func (r *Rows) Close() error {
	if r.body == nil {
		return nil
	}
	if _, err := io.ReadAll(r.body); err != nil {
		return err
	}
	return r.body.Close()
}

func (r *Rows) Scan(target interface{}) error {
	if err := r.dec.Decode(target); err != nil {
		return err
	}
	r.count++
	return nil
}

func (r *Rows) Next() bool {
	if r.dec != nil && r.dec.More() {
		return true
	}
	url, ok := r.nextpage(r.count)
	if !ok {
		return false
	}
	return r.next(url)
}

func (r *Rows) next(url string) bool {
	if err := r.Close(); err != nil {
		r.err = err
		return false
	}

	r.body, r.err = r.provider(r.ctx, url)
	if r.err != nil || r.body == nil {
		return false
	}

	r.dec = json.NewDecoder(r.body)
	r.count = 0

	if ok, err := r.seek(); err != nil || !ok {
		r.err = err
		return false
	}

	if ok, err := r.ensure(); err != nil || !ok {
		r.err = err
		return false
	}

	return r.dec.More()
}

func (r *Rows) ensure() (bool, error) {
	token, err := r.dec.Token()
	if err != nil {
		return false, err
	}
	if token != tokenArrStart {
		return false, fmt.Errorf("expected [ at %d position; got: %v", r.dec.InputOffset(), token)
	}
	return true, nil
}

func (r *Rows) seek() (bool, error) {
	if r.level == 0 {
		return true, nil
	}
	var level int
	for r.dec.More() || level != 0 {
		token, err := r.dec.Token()
		if err != nil {
			return false, err
		}
		switch {
		case token == r.token && r.level == level:
			return true, nil
		case token == tokenArrStart || token == tokenObjStart:
			level++
		case token == tokenArrFinish || token == tokenObjFinish:
			level--
		}
	}

	return false, nil
}
