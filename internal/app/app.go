package app

import (
	"context"
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
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/vyacheslavskl/go-shorterner/internal/auth"
	"github.com/vyacheslavskl/go-shorterner/internal/config"
	"github.com/vyacheslavskl/go-shorterner/internal/handler"
	"github.com/vyacheslavskl/go-shorterner/internal/repository"
	"github.com/vyacheslavskl/go-shorterner/internal/service"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const taskSize = 1000

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

	jwt := os.Getenv("JWT_KEY")
	if jwt == "" {
		jwt = "secret_temp_key"
	}
	jwtService := auth.NewJWTService([]byte(jwt))

	flag.Var(&addr.Address, "a", "Net address host:port")
	flag.Var(&addr.RedirectAddress, "b", "Net address host:port")
	flag.Parse()

	cfg := config.GetConfig(addr)
	sugar.Infow("Started with config",
		"Address", cfg.Address.String(),
		"RedirectAddress", cfg.RedirectAddress.String(),
		"StoragePath", storagePath,
	)
	var pool *pgxpool.Pool
	if dsn != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		pool, err = pgxpool.New(ctx, dsn)
		if err != nil {
			sugar.Errorln("Unable to connect to database ", err)
			return err
		}
	} else {
		pool = nil
	}

	if pool != nil {
		db := stdlib.OpenDB(*pool.Config().ConnConfig)
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

		m, err := migrate.NewWithDatabaseInstance(src, "postgres", driver)
		if err != nil {
			sugar.Errorln("Migrations failed", err)
			return err
		}
		m.Up()
		sugar.Infow("Migrations applied")
	}

	var repo repository.Repository
	if pool != nil {
		repo, err = repository.NewDBRepo(pool)
		if err != nil {
			return err
		}
	} else {
		repo, err = repository.NewMapRepo(nil, storagePath)
		if err != nil {
			return err
		}
	}

	srv := service.NewService(repo)

	deleteTaskCh := make(chan handler.DeleteTask, taskSize)
	go func() {
		for task := range deleteTaskCh {
			if err := srv.DeleteUserUrls(task.Context, task.UserID, task.ShortUrls); err != nil {
				sugar.Warnw("delete failed", "error", err, "userID", task.UserID)
			}
		}
		sugar.Infow("Delete worker stopped")
	}()

	handler := handler.NewShorterHandler(cfg, srv, sugar, jwtService, deleteTaskCh)

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
				close(deleteTaskCh)
				repo.SaveData()
				repo.Close(shutdownCtx)
				sugar.Infoln("Graceful shutdown")
				return nil
			}
		case err, ok := <-errCh:
			if !ok {
				continue
			}
			sugar.Infoln("Error when saving data:", err)
			return err
		}
	}
}
