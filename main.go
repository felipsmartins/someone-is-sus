package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/felipsmartins/someone-is-sus/internal/database"
	"github.com/felipsmartins/someone-is-sus/internal/steam"
	"github.com/felipsmartins/someone-is-sus/internal/types/httpmessage"
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
	steamID, err := steamClient.GetSteamIDByCustomURL(profileURL)

	if err != nil {
		hs.logger.Error(fmt.Sprintf("report_failed: error requesting steam API"), "detail", err, "profile_url", profileURL, "steam_id", steamID)
		return
	}

	payload, err := io.ReadAll(r.Body)

	if err != nil {
		hs.logger.Error(fmt.Sprintf("report_failed: error reading request body"),
			"detail", err,
			"profile_url", profileURL,
			"steam_id", steamID,
			"payload", payload)

		resBody, _ := json.Marshal(map[string]any{
			"error": "error reading request body",
		})

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write(resBody)

		return
	}
	// parse json body
	var reportInfo httpmessage.ReportData
	err = json.Unmarshal(payload, &reportInfo)

	if err != nil {
		hs.logger.Error(fmt.Sprintf("report_failed: error unmarshaling request body: %v", err),
			"detail", err, "payload", string(payload))

		resBody, _ := json.Marshal(map[string]any{
			"error": "error parsing request body",
		})

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write(resBody)

		return
	}

	if err = RegisterReportedPlayer(r.Context(), steamID, reportInfo.GameID, reportInfo.ReportedBy); err != nil {
		hs.logger.Error(fmt.Sprintf("report_failed: error saving"), "detail", err, "profile_url",
			profileURL, "steam_id", steamID)

		resBody, _ := json.Marshal(map[string]any{
			"error": "error saving into storage",
		})

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write(resBody)

		return
	}

	res, _ := json.Marshal(map[string]any{
		"steam_id": steamID,
	})

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(res)

	hs.logger.Info("reporting profile URL", "profile", steamID)
}

func GetDatabaseConnection() (*sql.DB, error) {
	connectionOptions := url.Values{}
	connectionOptions.Add("mode", "rw")
	connectionOptions.Add("_foreign_keys", "on")
	dsn := fmt.Sprintf("file:./sus.sqlite?%s", connectionOptions.Encode())

	fmt.Printf("dsn: %v\n", dsn)

	db, err := sql.Open("sqlite3", dsn)

	if err != nil {
		return nil, fmt.Errorf("connecting to database. DSN: %s. Error details: %w", dsn, err)
	}

	return db, nil
}

func RegisterReportedPlayer(ctx context.Context, reportedPlayerID string, gameID int64, reportedBy string) error {
	conn, err := GetDatabaseConnection()

	if err != nil {
		return err
	}

	defer conn.Close()

	queries := database.New(conn)
	_, err = queries.RegisterPlayer(ctx, database.RegisterPlayerParams{
		PlayerID:   reportedPlayerID,
		GameID:     gameID,
		ReportedBy: sql.NullString{String: reportedBy, Valid: true},
		ReportedAt: time.Now().UTC().Format(time.RFC3339),
	})

	if err != nil {
		return err
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
