package app

import (
	"context"
	"encoding/json"
	"flag"
	"maps"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/vyacheslavskl/go-shorterner/internal/config"
	"github.com/vyacheslavskl/go-shorterner/internal/handler"
	"github.com/vyacheslavskl/go-shorterner/internal/repository"
	"github.com/vyacheslavskl/go-shorterner/internal/service"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func saveStorage(filename string, m map[string]string) error {
	existing := make(map[string]string)
	if _, err := os.Stat(filename); err == nil {
		_ = loadStorage(filename, &existing)
	}

	maps.Copy(m, existing)

	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(m)
}

func loadStorage(filename string, m *map[string]string) error {
	file, err := os.Open(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer file.Close()

	return json.NewDecoder(file).Decode(m)
}

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

	storagePath := os.Getenv("FILE_STORAGE_PATH")
	if storagePath == "" {
		flag.StringVar(&storagePath, "f", "urls.json", "path to storage urls")
	}

	flag.Var(&addr.Address, "a", "Net address host:port")
	flag.Var(&addr.RedirectAddress, "b", "Net address host:port")
	flag.Parse()

	cfg := config.GetConfig(addr)
	sugar.Infow("Started with config",
		"Address", cfg.Address.String(),
		"RedirectAddress", cfg.RedirectAddress.String(),
		"StoragePath", storagePath,
	)

	repo := repository.NewMapRepo()
	err = loadStorage(storagePath, repo.LoadData())
	if err != nil {
		sugar.Warnln("Can not load Data storage")
	}

	srv := service.NewService(repo)

	handler := handler.NewShorterHandler(cfg, srv, sugar)

	server := &http.Server{
		Addr:    cfg.Address.String(),
		Handler: handler.Routes(),
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			sugar.Errorln(err.Error())
		}
	}()

	<-stop
	shutdownCtx := context.Background()
	if err := server.Shutdown(shutdownCtx); err != nil {
		return err
	} else {
		saveStorage(storagePath, repo.StoreData())
		sugar.Infoln("Graceful shutdown")
		return nil
	}
}
