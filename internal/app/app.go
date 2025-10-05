package app

import (
	"net/http"
	"github.com/vyacheslavskl/go-shorterner/internal/handler"
)

type App struct {
	routes *http.ServeMux
}

func New() *App {
	app := &App{routes: http.NewServeMux()}
	app.routes.HandleFunc("/", handler.PostRoot)
	return app
}

func (a *App) Run() error {
	err := http.ListenAndServe(":8080", a.routes)
	if err != nil {
		return err
	}
	return nil
}
