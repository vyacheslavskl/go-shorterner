package main

import "github.com/vyacheslavskl/go-shorterner/internal/app"

func main() {
	if err := app.Run(); err != nil {
		panic(err)
	}
}
