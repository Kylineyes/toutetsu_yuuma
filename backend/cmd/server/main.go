package main

import (
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"

	"github.com/Kylineyes/toutetsu_yuuma/backend/internal/api"
	"github.com/Kylineyes/toutetsu_yuuma/backend/internal/config"
	"github.com/Kylineyes/toutetsu_yuuma/backend/internal/db"
	"github.com/Kylineyes/toutetsu_yuuma/backend/internal/pipeline"
	"github.com/Kylineyes/toutetsu_yuuma/backend/internal/worker"
)

func main() {
	cfg := config.Load()
	for _, d := range []string{cfg.DataDir, filepath.Join(cfg.DataDir, "db")} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			log.Fatalf("mkdir %s: %v", d, err)
		}
	}
	database, err := db.Open(filepath.Join(cfg.DataDir, "db", "app.db"))
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	if err := database.Migrate(); err != nil {
		log.Fatalf("migrate: %v", err)
	}
	runner := &pipeline.Runner{DB: database, Cfg: cfg}
	wk := worker.New(runner, 2)
	a := &api.API{DB: database, Cfg: cfg, Worker: wk}

	addr := net.JoinHostPort(cfg.Host, cfg.Port)
	log.Printf("toutetsu_yuuma backend listening on %s (data=%s)", addr, cfg.DataDir)
	if err := http.ListenAndServe(addr, a.Routes()); err != nil {
		log.Fatalf("server: %v", err)
	}
}
