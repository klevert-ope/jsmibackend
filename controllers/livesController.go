package controllers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"jsmi-api/db"
	"jsmi-api/middlewares"
	"jsmi-api/models"
	"jsmi-api/validation"
	"net/http"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

func SetupLiveRoutes(r *mux.Router) {
	livesRouter := r.PathPrefix("/lives").Subrouter()
	livesRouter.HandleFunc("", GetLives).Methods("GET")
	livesRouter.HandleFunc("", GetLive).Methods("GET").Queries("id", "{id}")
	livesRouter.HandleFunc("", CreateLive).Methods("POST")
	livesRouter.HandleFunc("", UpdateLive).Methods("PUT").Queries("id", "{id}")
	livesRouter.HandleFunc("", DeleteLive).Methods("DELETE").Queries("id", "{id}")
}

func GetLives(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id != "" {
		GetLive(w, r)
		return
	}

	ctx := r.Context()
	lives, err := fetchLives(ctx)
	if err != nil {
		middlewares.HttpError(w, "Failed to fetch lives", http.StatusInternalServerError, err)
		return
	}

	middlewares.RespondJSON(w, lives, http.StatusOK)
}

func fetchLives(ctx context.Context) ([]models.Live, error) {
	cachedData, err := db.RedisClient.Get(ctx, "lives").Result()
	if err == nil {
		var lives []models.Live
		if err := json.Unmarshal([]byte(cachedData), &lives); err != nil {
			return nil, fmt.Errorf("error unmarshalling cached lives data: %w", err)
		}
		return lives, nil
	} else if !errors.Is(err, redis.Nil) {
		return nil, fmt.Errorf("error fetching lives from Redis cache: %w", err)
	}

	rows, err := db.DB.QueryContext(ctx, "SELECT id, title, link, created_at FROM lives")
	if err != nil {
		return nil, fmt.Errorf("error querying database: %w", err)
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			err = fmt.Errorf("error closing rows: %w", closeErr)
		}
	}()

	var lives []models.Live
	for rows.Next() {
		var live models.Live
		if err := rows.Scan(&live.ID, &live.Title, &live.Link, &live.CreatedAt); err != nil {
			return nil, fmt.Errorf("error scanning row: %w", err)
		}
		lives = append(lives, live)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over rows: %w", err)
	}

	jsonData, err := json.Marshal(lives)
	if err == nil {
		const CacheTime = 7 * 24 * time.Hour
		err = db.RedisClient.Set(ctx, "lives", jsonData, CacheTime).Err()
		if err != nil {
			return nil, fmt.Errorf("error setting lives cache: %w", err)
		}
	}

	return lives, nil
}

func GetLive(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")

	if idStr == "" {
		http.Error(w, "ID parameter is required", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	live, err := fetchLive(ctx, idStr)

	if err != nil {
		middlewares.HttpError(w, "Live not found", http.StatusNotFound, err)
		return
	}

	middlewares.RespondJSON(w, live, http.StatusOK)
}

func fetchLive(ctx context.Context, liveID string) (models.Live, error) {
	cachedData, err := db.RedisClient.Get(ctx, "live:"+liveID).Result()
	if err == nil {
		var live models.Live
		if err := json.Unmarshal([]byte(cachedData), &live); err != nil {
			return models.Live{}, fmt.Errorf("error unmarshalling cached live data: %w", err)
		}
		return live, nil
	} else if !errors.Is(err, redis.Nil) {
		return models.Live{}, fmt.Errorf("error fetching live %s from Redis cache: %w", liveID, err)
	}

	var live models.Live
	err = db.DB.QueryRowContext(ctx, "SELECT id, title, link, created_at FROM lives WHERE id = $1", liveID).
		Scan(&live.ID, &live.Title, &live.Link, &live.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.Live{}, fmt.Errorf("live %s not found: %w", liveID, sql.ErrNoRows)
		}
		return models.Live{}, fmt.Errorf("error querying database: %w", err)
	}

	jsonData, err := json.Marshal(live)
	if err == nil {
		const CacheTime = 7 * 24 * time.Hour
		err = db.RedisClient.Set(ctx, "live:"+liveID, jsonData, CacheTime).Err()
		if err != nil {
			return models.Live{}, fmt.Errorf("error setting live cache: %w", err)
		}
	}

	return live, nil
}

func CreateLive(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var live models.Live
	if err := json.NewDecoder(r.Body).Decode(&live); err != nil {
		middlewares.HttpError(w, "Invalid JSON payload", http.StatusBadRequest, err)
		return
	}

	if err := validation.ValidateLives(live); err != nil {
		middlewares.HttpError(w, err.Error(), http.StatusBadRequest, err)
		return
	}

	live.ID = uuid.New()
	live.CreatedAt = time.Now()

	if err := insertLive(ctx, live); err != nil {
		middlewares.HttpError(w, "Failed to create live", http.StatusInternalServerError, err)
		return
	}

	err := db.RedisClient.Del(ctx, "lives").Err()
	if err != nil {
		middlewares.HttpError(w, "Failed to clear lives cache", http.StatusInternalServerError, err)
		return
	}

	middlewares.RespondJSON(w, live, http.StatusCreated)
}

func insertLive(ctx context.Context, live models.Live) error {
	_, err := db.DB.ExecContext(ctx, "INSERT INTO lives (id, title, link, created_at) VALUES ($1, $2, $3, $4)",
		live.ID, live.Title, live.Link, live.CreatedAt)
	return err
}

func UpdateLive(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		http.Error(w, "Live ID is required", http.StatusBadRequest)
		return
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		middlewares.HttpError(w, "Invalid ID parameter", http.StatusBadRequest, err)
		return
	}

	var live models.Live
	if err := json.NewDecoder(r.Body).Decode(&live); err != nil {
		middlewares.HttpError(w, "Invalid JSON payload", http.StatusBadRequest, err)
		return
	}

	if err := validation.ValidateLives(live); err != nil {
		middlewares.HttpError(w, err.Error(), http.StatusBadRequest, err)
		return
	}

	live.ID = id

	if err := updateLive(ctx, live); err != nil {
		middlewares.HttpError(w, "Failed to update live", http.StatusInternalServerError, err)
		return
	}

	err = db.RedisClient.Del(ctx, "live:"+idStr).Err()
	if err != nil {
		middlewares.HttpError(w, "Failed to clear live cache", http.StatusInternalServerError, err)
		return
	}

	err = db.RedisClient.Del(ctx, "lives").Err()
	if err != nil {
		middlewares.HttpError(w, "Failed to clear lives cache", http.StatusInternalServerError, err)
		return
	}

	middlewares.RespondJSON(w, live, http.StatusOK)
}

func updateLive(ctx context.Context, live models.Live) error {
	_, err := db.DB.ExecContext(ctx, "UPDATE lives SET title=$1, link=$2 WHERE id=$3",
		live.Title, live.Link, live.ID)
	return err
}

func DeleteLive(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		http.Error(w, "Live ID is required", http.StatusBadRequest)
		return
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		middlewares.HttpError(w, "Invalid ID parameter", http.StatusBadRequest, err)
		return
	}

	if err := deleteLive(ctx, id); err != nil {
		middlewares.HttpError(w, "Failed to delete live", http.StatusInternalServerError, err)
		return
	}

	err = db.RedisClient.Del(ctx, "live:"+idStr).Err()
	if err != nil {
		middlewares.HttpError(w, "Failed to clear live cache", http.StatusInternalServerError, err)
		return
	}

	err = db.RedisClient.Del(ctx, "lives").Err()
	if err != nil {
		middlewares.HttpError(w, "Failed to clear lives cache", http.StatusInternalServerError, err)
		return
	}

	middlewares.RespondJSON(w, map[string]string{"message": "Live deleted"}, http.StatusOK)
}

func deleteLive(ctx context.Context, id uuid.UUID) error {
	_, err := db.DB.ExecContext(ctx, "DELETE FROM lives WHERE id=$1", id)
	return err
}
