package app

import (
	"flag"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/vyacheslavskl/go-shorterner/internal/config"
	"github.com/vyacheslavskl/go-shorterner/internal/handler"
)

type App struct {
	routes *chi.Mux
	config *config.Config
}

func Run() error {

	addr := new(config.Config)
	flag.Var(&addr.Address, "a", "Net address host:port")
	flag.Var(&addr.RedirectAddress, "b", "Net address host:port")
	flag.Parse()

	cfg := config.GetConfig(addr)

	app := &App{routes: chi.NewMux(), config: cfg}

	app.routes.Get("/", handler.PostRootHandler(cfg))
	app.routes.Get("/{id}", handler.PostRootHandler(cfg))
	app.routes.Post("/", handler.PostRootHandler(cfg))

	err := http.ListenAndServe(app.config.Address.String(), app.routes)
	if err != nil {
		return err
	}
	return nil
}
