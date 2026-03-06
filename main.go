package main

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"proxy/config"
	"proxy/health"
	"proxy/internal/logger"
	"proxy/internal/proxy"
	"proxy/internal/sys"
)

func main() {
	// -- Load config ---------------------------------------------------------
	if len(os.Args) < 2 {
		fmt.Printf("Usage: %s <config-file>\n", os.Args[0])
		os.Exit(1)
	}
	cfgPath := os.Args[1]

	debug := os.Getenv("DEBUG") == "true"
	structured := os.Getenv("LOG_FORMAT") == "json"
	logger.Init(debug, structured)

	cfg, err := config.LoadConfig(cfgPath)
	if err != nil {
		logger.Fatal("failed to load config", "error", err)
	}

	// -- System tuning -------------------------------------------------------
	sys.Tuning()
	go sys.PeriodicGC()

	// -- Proxy initialization ------------------------------------------------
	proxy.Init(cfg)

	// -- Health initialization -----------------------------------------------
	health.SetConfigVersion(cfg)

	// -- Routes --------------------------------------------------------------
	mux := http.NewServeMux()
	for _, r := range cfg.Routes {
		t, err := url.Parse(r.Target)
		if err != nil {
			logger.Fatal("bad target", "target", r.Target, "error", err)
		}
		mux.HandleFunc(r.Path, proxy.Handler(r.Path, t))
		logger.Info("route configured", "path", r.Path, "target", r.Target)
	}
	mux.HandleFunc("/health", health.HandleHealth)

	// -- Server --------------------------------------------------------------
	srv := &http.Server{
		Addr:           cfg.Listen,
		Handler:        proxy.CORSMiddleware(mux),
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   120 * time.Second,
		IdleTimeout:    30 * time.Second,
		MaxHeaderBytes: 1 << 16,
	}

	// -- Graceful shutdown ---------------------------------------------------
	ctx, stop := signal.NotifyContext(context.Background(),
		syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		logger.Info("shutting down...")
		c, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		srv.Shutdown(c)
	}()

	// -- Start ---------------------------------------------------------------
	logger.Info("listening", "addr", cfg.Listen)
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		logger.Fatal("server error", "error", err)
	}
}
