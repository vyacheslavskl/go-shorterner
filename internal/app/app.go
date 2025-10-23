package app

import (
	"flag"
	"net/http"
	"os"

	"github.com/vyacheslavskl/go-shorterner/internal/config"
	"github.com/vyacheslavskl/go-shorterner/internal/handler"
	"github.com/vyacheslavskl/go-shorterner/internal/repository"
	"github.com/vyacheslavskl/go-shorterner/internal/service"
)

func Run() error {
	addr := &config.Config{
		Address:         config.NetAddress{Protocol: "", Host: "localhost", Port: 8080},
		RedirectAddress: config.NetAddress{Protocol: "http://", Host: "localhost", Port: 8080},
	}

	serAdr := os.Getenv("SERVER_ADDRESS")
	if serAdr != "" {
		_ = addr.Address.Set(serAdr)
	}

	basAdr := os.Getenv("BASE_URL")
	if basAdr != "" {
		_ = addr.RedirectAddress.Set(basAdr)
	}

	flag.Var(&addr.Address, "a", "Net address host:port")
	flag.Var(&addr.RedirectAddress, "b", "Net address host:port")
	flag.Parse()

	cfg := config.GetConfig(addr)
	repo := repository.NewMapRepo()
	srv := service.NewService(repo)

	handler := handler.NewShorterHandler(cfg, srv)

	err := http.ListenAndServe(cfg.Address.String(), handler.Routes())
	if err != nil {
		return err
	}
	return nil
}
