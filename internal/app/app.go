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

	ser_adr := os.Getenv("SERVER_ADDRESS")
	if ser_adr != "" {
		_ = addr.Address.Set(ser_adr)
	}

	bas_adr := os.Getenv("BASE_URL")
	if bas_adr != "" {
		_ = addr.RedirectAddress.Set(ser_adr)
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
