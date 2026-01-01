package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/felipsmartins/someone-is-sus/internal/database"
	"github.com/felipsmartins/someone-is-sus/internal/steam"
	_ "github.com/mattn/go-sqlite3"
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
	hs.logger.Debug("user endpoint called")
	profileURL := r.URL.Query().Get("url")
	steamClient := steam.New(os.Getenv("STEAM_API_KEY"))
	val, err := steamClient.GetSteamIDByCustomURL(profileURL)

	if err != nil {
		hs.logger.Error(fmt.Sprintf("report_failed: error requesting steam API"), "detail", err, "profile", val)
		return
	}

	if err = reportPlayer(r.Context()); err != nil {
		hs.logger.Error(fmt.Sprintf("report_failed: error saving"), "detail", err, "profile", val)
		return
	}

	_, _ = w.Write([]byte("\nsteamID:" + val))

	hs.logger.Info("reporting profile URL", "profile", val)
}

func reportPlayer(ctx context.Context) error {
	db, err := sql.Open("sqlite3", "./sus.sqlite")

	if err != nil {
		return fmt.Errorf("connecting database: %w", err)
	}

	defer db.Close()

	queries := database.New(db)
	token := rand.Text()
	_, err = queries.RegisterPlayer(ctx, database.RegisterPlayerParams{
		PlayerID:   token,
		GameID:     1,
		ReportedBy: sql.NullString{String: "@some", Valid: true},
		ReportedAt: time.Now().UTC().Format(time.RFC3339),
	})

	if err != nil {
		log.Fatal(err)
	}

	return nil
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
