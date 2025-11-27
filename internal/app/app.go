package app

import (
	"context"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
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

	// fullPath := os.Getenv("MIGRATIONS_PATH")
	// if dsn == "" {
	// 	flag.StringVar(&fullPath, "m", "/migrations", "migration path")
	// }	

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

	if conn != nil {
		db, err := sql.Open("postgres", dsn)
		if err != nil {
			return err
		}
		driver, err := postgres.WithInstance(db, &postgres.Config{})
		if err != nil {
			return err
		}
		_, filename, _, ok := runtime.Caller(0)
		if !ok {
			return errors.New("failed to get runtime caller")
		}
		path := filepath.Dir(filename)
		fullPath := filepath.Join(path, "..", "..", "migrations")
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			return fmt.Errorf("migrations directory not found: %s", fullPath)
		}
		src := "file://" + filepath.ToSlash(fullPath)
		sugar.Errorln("Migrations path", src)

		m, err := migrate.NewWithDatabaseInstance(src, "postgres", driver)
		if err != nil {
			sugar.Errorln("Migrations failed", err)
			return err
		}
		m.Up()
		sugar.Infow("Migrations applied")
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
