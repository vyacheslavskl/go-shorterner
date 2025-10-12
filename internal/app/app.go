package app

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/vyacheslavskl/go-shorterner/internal/handler"
)

type App struct {
	routes *chi.Mux
}

func New() *App {
	app := &App{routes: chi.NewMux()}

	app.routes.Get("/", handler.PostRoot)
	app.routes.Get("/{id}", handler.PostRoot)
	app.routes.Post("/", handler.PostRoot)
	return app
}

func (a *App) Run() error {
	err := http.ListenAndServe(":8080", a.routes)
	if err != nil {
		return err
	}
	return nil
}
