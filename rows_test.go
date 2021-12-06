package apimono

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

func Test_Rows_Full(t *testing.T) {
	data := make([]struct {
		ID int `json:"id"`
	}, 1000)
	for i := 0; i < len(data); i++ {
		data[i].ID = i
	}
	buffer := bytes.NewBuffer(make([]byte, 0, 1<<14))
	if err := json.NewEncoder(buffer).Encode(data); err != nil {
		t.Fatal(err)
	}

	rows := Rows{
		ctx: context.Background(),
		provider: func(c context.Context, s string) (io.ReadCloser, error) {
			return io.NopCloser(buffer), nil
		},
		nextpage: func(i int) (string, bool) {
			return "", i == 0
		},
	}

	var total int
	for rows.Next() {
		var item struct {
			ID int `json:"id"`
		}
		if err := rows.Scan(&item); err != nil {
			t.Fatal(err)
		}
		if item.ID != total {
			t.Fatalf("unexpectet result; exp: %d; got: %d", total, item.ID)
		}
		total++
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err, total)
	}
	if err := rows.Close(); err != nil {
		t.Fatal(err, total)
	}
	if total != len(data) {
		t.Fatalf("unexpected number of items; expected: %d; got: %d", len(data), total)
	}
}

type mockReadClose struct {
	read, close error
}

func (r mockReadClose) Close() error {
	return r.close
}

func (r mockReadClose) Read([]byte) (int, error) {
	return 0, r.read
}

func Test_Rows_Close(t *testing.T) {
	testCases := []struct {
		name      string
		body      io.ReadCloser
		shouldErr bool
	}{
		{
			name:      "error on read",
			shouldErr: true,
			body:      mockReadClose{read: io.ErrClosedPipe},
		},
		{
			name:      "error on close",
			shouldErr: true,
			body: mockReadClose{
				read:  io.EOF,
				close: io.ErrClosedPipe,
			},
		},
		{
			name:      "no errors",
			shouldErr: false,
			body:      io.NopCloser(http.NoBody),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			r := Rows{
				body: tc.body,
			}
			err := r.Close()
			if tc.shouldErr && err == nil {
				t.Fatal("expected error")
			}
			if !tc.shouldErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func Test_Rows_Scan(t *testing.T) {
	testCases := []struct {
		name      string
		body      string
		shouldErr bool
		result    interface{}
		target    interface{}
	}{
		{
			name:   "correct case",
			result: "hello",
			target: new(string),
			body:   "\"hello\"",
		},
		{
			name:      "incorrect target",
			result:    "hello",
			target:    new(int),
			body:      "\"hello\"",
			shouldErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			r := Rows{
				dec: json.NewDecoder(strings.NewReader(tc.body)),
			}
			err := r.Scan(tc.target)
			if tc.shouldErr && err == nil {
				t.Fatal("expected error")
			}
			if !tc.shouldErr {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if reflect.DeepEqual(tc.result, tc.target) {
					t.Fatalf("unexpected result\n\texpected: %#v\n\tgot: %#v", tc.result, tc.target)
				}
			}
		})
	}
}

func Test_Rows_next(t *testing.T) {
	testCases := []struct {
		name      string
		shouldErr bool
		ok        bool
		level     int
		token     json.Token
		body      io.ReadCloser
		provider  func(context.Context, string) (io.ReadCloser, error)
	}{
		{
			ok:        false,
			shouldErr: true,
			name:      "error on close",
			body:      mockReadClose{read: io.ErrClosedPipe},
		},
		{
			ok:        false,
			shouldErr: true,
			name:      "provider error",
			body:      http.NoBody,
			provider: func(context.Context, string) (io.ReadCloser, error) {
				return nil, context.Canceled
			},
		},
		{
			ok:        false,
			shouldErr: true,
			name:      "seek error",
			level:     1,
			body:      http.NoBody,
			provider: func(context.Context, string) (io.ReadCloser, error) {
				return io.NopCloser(
					strings.NewReader("{="),
				), nil
			},
		},
		{
			ok:    false,
			name:  "seek false",
			level: 1,
			body:  http.NoBody,
			provider: func(context.Context, string) (io.ReadCloser, error) {
				return io.NopCloser(
					strings.NewReader("{}"),
				), nil
			},
		},
		{
			ok:        false,
			shouldErr: true,
			name:      "ensure error",
			level:     1,
			token:     "items",
			body:      http.NoBody,
			provider: func(context.Context, string) (io.ReadCloser, error) {
				return io.NopCloser(
					strings.NewReader("{\"items\":=}"),
				), nil
			},
		},
		{
			ok:        false,
			shouldErr: true,
			name:      "ensure not array",
			level:     1,
			token:     "items",
			body:      http.NoBody,
			provider: func(context.Context, string) (io.ReadCloser, error) {
				return io.NopCloser(
					strings.NewReader("{\"items\":{}}"),
				), nil
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rows := Rows{
				level:    tc.level,
				token:    tc.token,
				body:     tc.body,
				provider: tc.provider,
			}

			ok := rows.next("")

			if tc.shouldErr && rows.Err() == nil {
				t.Fatal("expected error")
			}
			if !tc.shouldErr {
				if err := rows.Err(); err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if tc.ok != ok {
					t.Fatalf("unexpected result: expected: %t; got: %t", tc.ok, ok)
				}
			}
		})
	}
}

func Test_Rows_ensure(t *testing.T) {
	testCases := []struct {
		name      string
		body      string
		shouldErr bool
		ok        bool
	}{
		{
			ok:        false,
			name:      "invalid token",
			body:      "=",
			shouldErr: true,
		},
		{
			ok:        false,
			name:      "not array token",
			body:      "{",
			shouldErr: true,
		},
		{
			ok:   true,
			name: "array token",
			body: "[",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			r := Rows{
				dec: json.NewDecoder(strings.NewReader(tc.body)),
			}
			ok, err := r.ensure()
			if tc.shouldErr && err == nil {
				t.Fatal("expected error")
			}
			if !tc.shouldErr {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if tc.ok != ok {
					t.Fatalf("unexpected result: expected: %t; got: %t", tc.ok, ok)
				}
			}
		})
	}
}

func Test_Rows_seek(t *testing.T) {
	testCases := []struct {
		name      string
		level     int
		token     json.Token
		body      string
		shouldErr bool
		ok        bool
	}{
		{
			ok:   true,
			name: "level 0",
			body: "[]",
		},
		{
			ok:    false,
			name:  "empty body",
			level: 1,
		},
		{
			ok:    false,
			name:  "empty json object",
			level: 1,
			token: "items",
			body:  "{}",
		},
		{
			ok:        false,
			name:      "bad json object",
			level:     1,
			token:     "items",
			body:      "{[",
			shouldErr: true,
		},
		{
			name:  "level 1",
			level: 1,
			ok:    true,
			token: "items",
			body:  "{\"items\":[]}",
		},
		{
			name:  "level 1",
			level: 1,
			ok:    true,
			token: "items",
			body:  "{\"error\": null, \"items\":[]}",
		},
		{
			name:  "level 2",
			level: 2,
			ok:    true,
			token: "items",
			body: `{
				"error": null,
				"result": {
					"date": "2021-11-28",
					"items": []
				}
			}`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			r := Rows{
				level: tc.level,
				token: tc.token,
				dec:   json.NewDecoder(strings.NewReader(tc.body)),
			}
			ok, err := r.seek()
			if tc.shouldErr && err == nil {
				t.Fatal("expected error")
			}
			if !tc.shouldErr {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if tc.ok != ok {
					t.Fatalf("unexpected result: expected: %t; got: %t", tc.ok, ok)
				}
			}
		})
	}
}
