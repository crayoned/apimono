package apimono

import (
	"context"
	"fmt"
	"io"
	"net/http"
)

func HTTPProvider() func(context.Context, string) (io.ReadCloser, error) {
	return func(ctx context.Context, url string) (io.ReadCloser, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
		if err != nil {
			return nil, err
		}
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}
		if res.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("wrong res.code: %d", res.StatusCode)
		}
		return res.Body, nil
	}
}
