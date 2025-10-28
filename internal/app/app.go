package app

import (
	"flag"
	"net/http"
	"os"

	"github.com/vyacheslavskl/go-shorterner/internal/config"
	"github.com/vyacheslavskl/go-shorterner/internal/handler"
	"github.com/vyacheslavskl/go-shorterner/internal/repository"
	"github.com/vyacheslavskl/go-shorterner/internal/service"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func Run() error {
	log := zap.NewDevelopmentConfig()
	log.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
	logger, err := log.Build()
	if err != nil {
		return err
	}
	defer logger.Sync()

	sugar := logger.Sugar()

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
	sugar.Infow("Started with config",
		"Address", cfg.Address.String(),
		"RedirectAddress", cfg.RedirectAddress.String(),
	)
	repo := repository.NewMapRepo()
	srv := service.NewService(repo)

	handler := handler.NewShorterHandler(cfg, srv, sugar)

	err = http.ListenAndServe(cfg.Address.String(), handler.Routes())
	if err != nil {
		return err
	}
	return nil
}
