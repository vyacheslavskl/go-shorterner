package app

import (
	"context"
	"errors"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5"
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
		err = addr.Address.Set(serAdr)
		if err != nil {
			return err
		}
	}
	basAdr := os.Getenv("BASE_URL")
	if basAdr != "" {
		err = addr.RedirectAddress.Set(basAdr)
		if err != nil {
			return err
		}
	}

	storagePath := os.Getenv("FILE_STORAGE_PATH")
	if storagePath == "" {
		flag.StringVar(&storagePath, "f", "urls.json", "path to storage urls")
	}

	dsn := os.Getenv("DATABASE_DSN")
	if dsn == "" {
		flag.StringVar(&dsn, "d", "", "dsn for Postgres")
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
	var conn *pgx.Conn
	if dsn != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    	defer cancel()
		conn, err = pgx.Connect(ctx, dsn)
		if err != nil {
			sugar.Errorln("Unable to connect to database ", err)
			return err
		}
	} else {
		conn = nil
	}

	repo, err := repository.NewMapRepo(conn, storagePath)
	if err != nil {
		return err
	}
	srv := service.NewService(repo)
	handler := handler.NewShorterHandler(cfg, srv, sugar)

	server := &http.Server{
		Addr:    cfg.Address.String(),
		Handler: handler.Routes(),
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	errCh := repo.PeriodicSave(10 * time.Second)

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			sugar.Errorln(err.Error())
		}
	}()

	for {
		select {
		case <-stop:
			shutdownCtx := context.Background()
			if err := server.Shutdown(shutdownCtx); err != nil {
				return err
			} else {
				repo.SaveData()
				repo.Close(context.Background())
				sugar.Infoln("Graceful shutdown")
				return nil
			}
		case err, ok := <-errCh:
			if !ok {
				sugar.Infoln("Err channel is closed")
				return errors.New("errCh channel is closed")
			}
			sugar.Infoln("Error when saving data:", err)
			return err
		}
	}

}
