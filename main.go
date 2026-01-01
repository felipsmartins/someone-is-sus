package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"slices"
	"strings"

	"log/slog"

	"github.com/felipsmartins/someone-is-sus/internal/steam"
)

func configureLogger(attrs []slog.Attr) (*slog.Logger, error) {
	level := &slog.LevelVar{}
	err := level.UnmarshalText([]byte(strings.TrimSpace(os.Getenv("LOG_LEVEL"))))

	if err != nil {
		return nil, fmt.Errorf("invalid LOG_LEVEL value: '%s'", os.Getenv("LOG_LEVEL"))
	}

	opts := &slog.HandlerOptions{
		AddSource: true,
		Level:     level,
	}
	defaultAttrs := slices.Concat([]slog.Attr{
		slog.String("environ", os.Getenv("ENVIRON")),
	}, attrs)
	handler := slog.NewJSONHandler(os.Stdout, opts).WithAttrs(defaultAttrs)
	logger := slog.New(handler)

	slog.SetDefault(logger)

	return logger, nil
}

type handlerSet struct {
	logger *slog.Logger
}

func (hs *handlerSet) home(w http.ResponseWriter, r *http.Request) {
	hs.logger.Info("index endpoint called")
}

func (hs *handlerSet) reportUser(w http.ResponseWriter, r *http.Request) {
	profileURL := r.URL.Query().Get("url")
	steamClient := steam.New(os.Getenv("STEAM_API_KEY"))
	val, err := steamClient.GetSteamIDByCustomURL(profileURL)

	if err != nil {
		hs.logger.Error(fmt.Sprintf("report user failed: "))
		return
	}

	_, _ = w.Write([]byte("\nsteamID:" + val))

	hs.logger.Info("reporting profile URL", "profile", val)
}

func main() {
	logger, err := configureLogger(nil)

	if err != nil {
		log.Fatal(err)
	}

	// HTTP setup
	handlers := handlerSet{logger}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /sus/report", handlers.reportUser)
	mux.HandleFunc("GET /index", handlers.home)

	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	// Start the HTTP server on port 8080
	logger.Info("Server started", "addr", server.Addr)
	log.Fatal(server.ListenAndServe())
}
