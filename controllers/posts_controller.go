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
	"net/http"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

func SetupPostRoutes(r *mux.Router) {
	postsRouter := r.PathPrefix("/posts").Subrouter()
	postsRouter.HandleFunc("", GetPosts).Methods("GET")
	postsRouter.HandleFunc("", GetPost).Methods("GET").Queries("id", "{id}")
	postsRouter.HandleFunc("", CreatePost).Methods("POST")
	postsRouter.HandleFunc("", UpdatePost).Methods("PUT").Queries("id", "{id}")
	postsRouter.HandleFunc("", DeletePost).Methods("DELETE").Queries("id", "{id}")
}

func GetPosts(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id != "" {
		GetPost(w, r)
		return
	}

	ctx := r.Context()
	posts, err := fetchPosts(ctx)
	if err != nil {
		middlewares.HttpError(w, "Failed to fetch posts", http.StatusInternalServerError, err)
		return
	}

	middlewares.RespondJSON(w, posts, http.StatusOK)
}

func fetchPosts(ctx context.Context) ([]models.Post, error) {
	cachedData, err := db.RedisClient.Get(ctx, "posts").Result()
	if err == nil {
		var posts []models.Post
		if err := json.Unmarshal([]byte(cachedData), &posts); err != nil {
			return nil, fmt.Errorf("error unmarshalling cached posts data: %w", err)
		}
		return posts, nil
	} else if !errors.Is(err, redis.Nil) {
		return nil, fmt.Errorf("error fetching posts from Redis cache: %w", err)
	}

	rows, err := db.DB.QueryContext(ctx, "SELECT id, title, excerpt, body, created_at FROM posts")
	if err != nil {
		return nil, fmt.Errorf("error querying database: %w", err)
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			err = fmt.Errorf("error closing rows: %w", closeErr)
		}
	}()

	var posts []models.Post
	for rows.Next() {
		var post models.Post
		if err := rows.Scan(&post.ID, &post.Title, &post.Excerpt, &post.Body, &post.CreatedAt); err != nil {
			return nil, fmt.Errorf("error scanning row: %w", err)
		}
		posts = append(posts, post)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over rows: %w", err)
	}

	jsonData, err := json.Marshal(posts)
	if err == nil {
		const CacheTime = 7 * 24 * time.Hour
		db.RedisClient.Set(ctx, "posts", jsonData, CacheTime)
	}

	return posts, nil
}

func GetPost(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		http.Error(w, "ID parameter is required", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	post, err := fetchPost(ctx, idStr)
	if err != nil {
		middlewares.HttpError(w, "Post not found", http.StatusNotFound, err)
		return
	}

	middlewares.RespondJSON(w, post, http.StatusOK)
}

func fetchPost(ctx context.Context, postID string) (models.Post, error) {
	cachedData, err := db.RedisClient.Get(ctx, "post:"+postID).Result()
	if err == nil {
		var post models.Post
		if err := json.Unmarshal([]byte(cachedData), &post); err != nil {
			return models.Post{}, fmt.Errorf("error unmarshalling cached post data: %w", err)
		}
		return post, nil
	} else if !errors.Is(err, redis.Nil) {
		return models.Post{}, fmt.Errorf("error fetching post %s from Redis cache: %w", postID, err)
	}

	var post models.Post
	err = db.DB.QueryRowContext(ctx, "SELECT id, title, excerpt, body, created_at FROM posts WHERE id = $1", postID).
		Scan(&post.ID, &post.Title, &post.Excerpt, &post.Body, &post.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.Post{}, fmt.Errorf("post %s not found: %w", postID, sql.ErrNoRows)
		}
		return models.Post{}, fmt.Errorf("error querying database: %w", err)
	}

	jsonData, err := json.Marshal(post)
	if err == nil {
		const CacheTime = 7 * 24 * time.Hour
		db.RedisClient.Set(ctx, "post:"+postID, jsonData, CacheTime)
	}

	return post, nil
}

func CreatePost(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var post models.Post
	if err := json.NewDecoder(r.Body).Decode(&post); err != nil {
		middlewares.HttpError(w, "Invalid JSON payload", http.StatusBadRequest, err)
		return
	}

	if err := middlewares.ValidatePost(post); err != nil {
		middlewares.HttpError(w, err.Error(), http.StatusBadRequest, err)
		return
	}

	post.ID = uuid.New()
	post.CreatedAt = time.Now()

	if err := insertPost(ctx, post); err != nil {
		middlewares.HttpError(w, "Failed to create post", http.StatusInternalServerError, err)
		return
	}

	db.RedisClient.Del(ctx, "posts")
	middlewares.RespondJSON(w, nil, http.StatusCreated)
}

func insertPost(ctx context.Context, post models.Post) error {
	_, err := db.DB.ExecContext(ctx, "INSERT INTO posts (id, title, excerpt, body, created_at) VALUES ($1, $2, $3, $4, $5)",
		post.ID, post.Title, post.Excerpt, post.Body, post.CreatedAt)
	return err
}

func UpdatePost(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		http.Error(w, "Post ID is required", http.StatusBadRequest)
		return
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		middlewares.HttpError(w, "Invalid ID parameter", http.StatusBadRequest, err)
		return
	}

	var post models.Post
	if err := json.NewDecoder(r.Body).Decode(&post); err != nil {
		middlewares.HttpError(w, "Invalid JSON payload", http.StatusBadRequest, err)
		return
	}

	if err := middlewares.ValidatePost(post); err != nil {
		middlewares.HttpError(w, err.Error(), http.StatusBadRequest, err)
		return
	}

	post.ID = id
	if err := updatePost(ctx, post); err != nil {
		middlewares.HttpError(w, "Failed to update post", http.StatusInternalServerError, err)
		return
	}

	db.RedisClient.Del(ctx, "post:"+idStr)
	db.RedisClient.Del(ctx, "posts")
	middlewares.RespondJSON(w, nil, http.StatusNoContent)
}

func updatePost(ctx context.Context, post models.Post) error {
	_, err := db.DB.ExecContext(ctx, "UPDATE posts SET title = $1, excerpt = $2, body = $3, created_at = $4 WHERE id = $5",
		post.Title, post.Excerpt, post.Body, post.CreatedAt, post.ID)
	return err
}

func DeletePost(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		http.Error(w, "Post ID is required", http.StatusBadRequest)
		return
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		middlewares.HttpError(w, "Invalid ID parameter", http.StatusBadRequest, err)
		return
	}

	if err := deletePost(ctx, id); err != nil {
		middlewares.HttpError(w, "Failed to delete post", http.StatusInternalServerError, err)
		return
	}

	db.RedisClient.Del(ctx, "post:"+idStr)
	db.RedisClient.Del(ctx, "posts")
	middlewares.RespondJSON(w, nil, http.StatusNoContent)
}

func deletePost(ctx context.Context, id uuid.UUID) error {
	_, err := db.DB.ExecContext(ctx, "DELETE FROM posts WHERE id = $1", id)
	return err
}
