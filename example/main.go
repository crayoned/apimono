package main

import (
	"context"
	"fmt"
	"os/signal"
	"syscall"

	"github.com/crayoned/apimono"
)

func main() {

	ctx, cancel := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT, syscall.SIGTERM,
	)
	defer cancel()

	if err := fromFS(ctx); err != nil {
		panic(err)
	}

	if err := fromHTTP(ctx); err != nil {
		panic(err)
	}
}

func fromFS(ctx context.Context) error {
	opts := []apimono.RowsOption{
		apimono.WithToken("items", 1),
		apimono.WithNext(apimono.ScanFolder("./example/testdata")),
		apimono.WithProvider(apimono.FileProvider()),
	}
	rows, err := apimono.Build(ctx, opts...)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var item struct {
			ID   int    `json:"id"`
			Root string `json:"root"`
		}
		if err := rows.Scan(&item); err != nil {
			return err
		}
		fmt.Println(item.ID, item.Root)
	}
	return rows.Err()
}

func fromHTTP(ctx context.Context) error {

	url := "https://jsonplaceholder.typicode.com/users?_page=%d&_limit=%d"
	limit, offset, initial := 4, 0, true

	opts := []apimono.RowsOption{
		// apimono.WithToken(json.Delim('['), 0), // defaults
		apimono.WithProvider(apimono.HTTPProvider()),
		apimono.WithNext(func(amount int) (string, bool) {
			if !initial && amount != limit {
				return "", false
			}
			initial = false
			offset += amount
			return fmt.Sprintf(url, offset/limit, limit), true
		}),
	}
	rows, err := apimono.Build(ctx, opts...)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var item struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		}
		if err := rows.Scan(&item); err != nil {
			return err
		}
		fmt.Println(item.ID, item.Name)
	}
	return rows.Err()
}
