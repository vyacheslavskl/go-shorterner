package app

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/vyacheslavskl/go-shorterner/internal/config"
	"github.com/vyacheslavskl/go-shorterner/internal/handler"
)

type App struct {
	routes *chi.Mux
	config *config.Config
}

func New() *App {

	cfg := config.GetConfig()

	app := &App{routes: chi.NewMux(), config: cfg}

	app.routes.Get("/", handler.PostRoot)
	app.routes.Get("/{id}", handler.PostRoot)
	app.routes.Post("/", handler.PostRoot)

	return app
}

func (a *App) Run() error {

	err := http.ListenAndServe(a.config.Address.String(), a.routes)
	if err != nil {
		return err
	}
	return nil
}
