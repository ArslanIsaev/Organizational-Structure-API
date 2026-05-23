package main

import (
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	_ "github.com/lib/pq"
	"github.com/pressly/goose/v3"
	"github.com/yourname/org-api/internal/config"
	"github.com/yourname/org-api/internal/db"
	"github.com/yourname/org-api/internal/handler"
	"github.com/yourname/org-api/internal/middleware"
	"github.com/yourname/org-api/internal/repository"
	"github.com/yourname/org-api/internal/service"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	cfg := config.Load()

	gormDB, err := db.Connect(cfg)
	if err != nil {
		logger.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}

	sqlDB, err := gormDB.DB()
	if err != nil {
		logger.Error("failed to get sql.DB", "error", err)
		os.Exit(1)
	}
	if err := runMigrations(sqlDB, logger); err != nil {
		logger.Error("failed to run migrations", "error", err)
		os.Exit(1)
	}

	deptRepo := repository.NewDepartmentRepository(gormDB)
	empRepo := repository.NewEmployeeRepository(gormDB)
	deptSvc := service.NewDepartmentService(deptRepo)
	empSvc := service.NewEmployeeService(empRepo, deptRepo)
	deptH := handler.NewDepartmentHandler(deptSvc)
	empH := handler.NewEmployeeHandler(empSvc)

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux, deptH, empH)
	loggedMux := middleware.Logging(logger)(mux)

	addr := fmt.Sprintf(":%s", cfg.ServerPort)
	srv := &http.Server{
		Addr:         addr,
		Handler:      loggedMux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	logger.Info("server starting", "addr", addr)
	if err := srv.ListenAndServe(); err != nil {
		logger.Error("server stopped", "error", err)
		os.Exit(1)
	}
}

func runMigrations(sqlDB *sql.DB, logger *slog.Logger) error {
	goose.SetLogger(goose.NopLogger())
	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}
	logger.Info("running migrations")
	return goose.Up(sqlDB, "migrations")
}
