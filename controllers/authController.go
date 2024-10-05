package controllers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"jsmi-api/db"
	"jsmi-api/middlewares"
	"jsmi-api/models"
	"jsmi-api/utils"
	"jsmi-api/validation"
	"net/http"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/gorilla/mux"
)

type AuthHandler struct {
	Config *db.Config
}

func (h *AuthHandler) SetupUserRoutes(r *mux.Router) {
	usersRouter := r.PathPrefix("/auth").Subrouter()
	usersRouter.HandleFunc("/register", h.Register).Methods("POST")
	usersRouter.HandleFunc("/login", h.Login).Methods("POST")
	usersRouter.HandleFunc("/logoff", h.Logoff).Methods("POST")
	usersRouter.HandleFunc("/delete-account", h.DeleteAccount).Methods("DELETE")
	usersRouter.Handle("/change-password", middlewares.TokenAuthMiddleware(http.HandlerFunc(h.ChangePassword))).Methods("POST")
	usersRouter.HandleFunc("/refresh-token", h.RefreshToken).Methods("POST")
}

func (h *AuthHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	var refreshTokenRequest struct {
		RefreshToken string `json:"refreshToken"`
	}

	if err := json.NewDecoder(r.Body).Decode(&refreshTokenRequest); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	claims, err := utils.ValidatePASETO(refreshTokenRequest.RefreshToken)
	if err != nil {
		http.Error(w, "Invalid refresh token", http.StatusUnauthorized)
		return
	}

	accessToken, err := utils.GeneratePASETO(claims.UserID, 15*time.Minute)
	if err != nil {
		http.Error(w, "Failed to generate new access token", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]string{
		"accessToken": accessToken,
	}); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var user models.User

	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := validation.ValidateUserData(user); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	if err := CreateUser(ctx, db.DB, &user); err != nil {
		http.Error(w, "Failed to create user", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func CreateUser(ctx context.Context, db *sql.DB, user *models.User) error {
	if err := user.HashPassword(); err != nil {
		return err
	}

	query := `INSERT INTO users (username, email, password) VALUES ($1, $2, $3) RETURNING id, created_at`
	err := db.QueryRowContext(ctx, query, user.Username, user.Email, user.Password).Scan(&user.ID, &user.CreatedAt)
	if err != nil {
		return errors.New("failed to insert user into database: " + err.Error())
	}

	if err := SetUserCache(ctx, user); err != nil {
		return errors.New("failed to set user cache: " + err.Error())
	}

	return nil
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var credentials struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&credentials); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	user, err := GetUserByUsername(ctx, db.DB, credentials.Username)
	if err != nil {
		http.Error(w, "Failed to retrieve user", http.StatusInternalServerError)
		return
	}
	if user == nil {
		http.Error(w, "Invalid username or password", http.StatusUnauthorized)
		return
	}

	if !user.CheckPassword(credentials.Password) {
		http.Error(w, "Invalid username or password", http.StatusUnauthorized)
		return
	}

	accessToken, err := utils.GeneratePASETO(user.ID, 15*time.Minute)
	if err != nil {
		http.Error(w, "Failed to generate access token", http.StatusInternalServerError)
		return
	}

	refreshToken, err := utils.GeneratePASETO(user.ID, 7*24*time.Hour)
	if err != nil {
		http.Error(w, "Failed to generate refresh token", http.StatusInternalServerError)
		return
	}

	setAuthCookies(w, accessToken, refreshToken)

	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]string{
		"accessToken":  accessToken,
		"refreshToken": refreshToken,
	}); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

func setAuthCookies(w http.ResponseWriter, accessToken, refreshToken string) {
	http.SetCookie(w, &http.Cookie{
		Name:     "access_token",
		Value:    accessToken,
		Expires:  time.Now().Add(15 * time.Minute),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		Path:     "/",
	})

	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    refreshToken,
		Expires:  time.Now().Add(7 * 24 * time.Hour),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		Path:     "/",
	})
}

func GetUserByUsername(ctx context.Context, db *sql.DB, username string) (*models.User, error) {
	user, err := GetUserCache(ctx, username)
	if err != nil {
		return nil, err
	}
	if user != nil {
		return user, nil
	}

	var userFromDB models.User
	query := `SELECT id, username, email, password, created_at FROM users WHERE username = $1`
	err = db.QueryRowContext(ctx, query, username).Scan(&userFromDB.ID, &userFromDB.Username, &userFromDB.Email, &userFromDB.Password, &userFromDB.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, errors.New("failed to query user by username: " + err.Error())
	}

	if err := SetUserCache(ctx, &userFromDB); err != nil {
		return nil, errors.New("failed to set user cache: " + err.Error())
	}

	return &userFromDB, nil
}

func (h *AuthHandler) Logoff(w http.ResponseWriter, _ *http.Request) {
	clearAuthCookies(w)
	w.WriteHeader(http.StatusOK)
}

func clearAuthCookies(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     "access_token",
		Value:    "",
		Expires:  time.Now().Add(-1 * time.Hour),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		Path:     "/",
	})

	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		Expires:  time.Now().Add(-1 * time.Hour),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		Path:     "/",
	})
}

func (h *AuthHandler) DeleteAccount(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("access_token")
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	claims, err := utils.ValidatePASETO(cookie.Value)
	if err != nil {
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	userID := claims.UserID
	if err := DeleteUser(r.Context(), db.DB, userID); err != nil {
		http.Error(w, "Failed to delete account", http.StatusInternalServerError)
		return
	}

	clearAuthCookies(w)
	w.WriteHeader(http.StatusOK)
}

func DeleteUser(ctx context.Context, db *sql.DB, userID int64) error {
	user, err := GetUserByID(ctx, db, userID)
	if err != nil {
		return err
	}
	if user == nil {
		return errors.New("user not found")
	}

	if err := DeleteUserCache(ctx, user.Username); err != nil {
		return errors.New("failed to delete user cache: " + err.Error())
	}

	_, err = db.ExecContext(ctx, `DELETE FROM users WHERE id = $1`, userID)
	if err != nil {
		return errors.New("failed to delete user: " + err.Error())
	}
	return nil
}

func (h *AuthHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	var data struct {
		OldPassword string `json:"old_password"`
		NewPassword string `json:"new_password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	cookie, err := r.Cookie("access_token")
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	claims, err := utils.ValidatePASETO(cookie.Value)
	if err != nil {
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	userID := claims.UserID
	ctx := r.Context()
	user, err := GetUserByID(ctx, db.DB, userID)
	if err != nil {
		http.Error(w, "Failed to retrieve user", http.StatusInternalServerError)
		return
	}
	if user == nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	if !user.CheckPassword(data.OldPassword) {
		http.Error(w, "Old password is incorrect", http.StatusUnauthorized)
		return
	}

	if err := validation.ValidatePasswordChange(data.OldPassword, data.NewPassword); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	user.Password = data.NewPassword
	if err := user.HashPassword(); err != nil {
		http.Error(w, "Failed to hash new password", http.StatusInternalServerError)
		return
	}

	if err := UpdateUserPassword(ctx, db.DB, userID, user.Password); err != nil {
		http.Error(w, "Failed to update password", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func GetUserByID(ctx context.Context, db *sql.DB, userID int64) (*models.User, error) {
	var user models.User
	query := `SELECT id, username, email, password, created_at FROM users WHERE id = $1`
	err := db.QueryRowContext(ctx, query, userID).Scan(&user.ID, &user.Username, &user.Email, &user.Password, &user.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, errors.New("failed to query user by ID: " + err.Error())
	}

	return &user, nil
}

func UpdateUserPassword(ctx context.Context, db *sql.DB, userID int64, hashedPassword string) error {
	user, err := GetUserByID(ctx, db, userID)
	if err != nil {
		return err
	}
	if user == nil {
		return errors.New("user not found")
	}

	if err := DeleteUserCache(ctx, user.Username); err != nil {
		return errors.New("failed to delete user cache: " + err.Error())
	}

	_, err = db.ExecContext(ctx, `UPDATE users SET password = $1 WHERE id = $2`, hashedPassword, userID)
	if err != nil {
		return errors.New("failed to update user password: " + err.Error())
	}
	return nil
}

func SetUserCache(ctx context.Context, user *models.User) error {
	data, err := json.Marshal(user)
	if err != nil {
		return err
	}

	const CacheTime = 7 * 24 * time.Hour
	return db.RedisClient.Set(ctx, "user:"+user.Username, data, CacheTime).Err()
}

func GetUserCache(ctx context.Context, username string) (*models.User, error) {
	data, err := db.RedisClient.Get(ctx, "user:"+username).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil
		}
		return nil, err
	}

	var user models.User
	err = json.Unmarshal(data, &user)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func DeleteUserCache(ctx context.Context, username string) error {
	return db.RedisClient.Del(ctx, "user:"+username).Err()
}
