package apimono

import (
	"context"
	"io"
	"os"
	"path"
)

func FileProvider() func(context.Context, string) (io.ReadCloser, error) {
	return func(ctx context.Context, path string) (io.ReadCloser, error) {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		return os.Open(path)
	}
}

func ScanFolder(folder string) func(int) (string, bool) {
	files, err := os.ReadDir(folder)
	if err != nil {
		return noopNext
	}
	return func(_ int) (string, bool) {
		for i := 0; i < len(files); i++ {
			if files[i].IsDir() {
				continue
			}
			file := files[i]
			files = files[i+1:]
			return path.Join(folder, file.Name()), true
		}
		return "", false
	}
}
