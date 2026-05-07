package handler

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rahulbattula415/DatasetVersionControl/internal/db"
	"golang.org/x/crypto/bcrypt"
)

type authRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type authResponse struct {
	Token string  `json:"token"`
	User  db.User `json:"user"`
}

func RegisterHandler(pool *pgxpool.Pool, jwtSecret string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req authRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httpError(w, "invalid request body", http.StatusBadRequest)
			return
		}
		req.Email = strings.ToLower(strings.TrimSpace(req.Email))
		if req.Email == "" || req.Password == "" {
			httpError(w, "email and password are required", http.StatusBadRequest)
			return
		}
		if len(req.Password) < 8 {
			httpError(w, "password must be at least 8 characters", http.StatusBadRequest)
			return
		}

		existing, err := db.GetUserByEmail(r.Context(), pool, req.Email)
		if err != nil {
			httpError(w, "database error", http.StatusInternalServerError)
			return
		}
		if existing != nil {
			httpError(w, "email already registered", http.StatusConflict)
			return
		}

		hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			httpError(w, "failed to hash password", http.StatusInternalServerError)
			return
		}

		user, err := db.CreateUser(r.Context(), pool, req.Email, string(hash))
		if err != nil {
			httpError(w, "failed to create user", http.StatusInternalServerError)
			return
		}

		token, err := signToken(user.ID, user.Email, jwtSecret)
		if err != nil {
			httpError(w, "failed to sign token", http.StatusInternalServerError)
			return
		}

		jsonResponse(w, http.StatusCreated, authResponse{Token: token, User: *user})
	}
}

func LoginHandler(pool *pgxpool.Pool, jwtSecret string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req authRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httpError(w, "invalid request body", http.StatusBadRequest)
			return
		}
		req.Email = strings.ToLower(strings.TrimSpace(req.Email))

		user, err := db.GetUserByEmail(r.Context(), pool, req.Email)
		if err != nil {
			httpError(w, "database error", http.StatusInternalServerError)
			return
		}
		// Same error for "not found" and "wrong password" — no user enumeration
		if user == nil || bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)) != nil {
			httpError(w, "invalid email or password", http.StatusUnauthorized)
			return
		}

		token, err := signToken(user.ID, user.Email, jwtSecret)
		if err != nil {
			httpError(w, "failed to sign token", http.StatusInternalServerError)
			return
		}

		jsonResponse(w, http.StatusOK, authResponse{Token: token, User: *user})
	}
}

func signToken(userID, email, secret string) (string, error) {
	claims := jwt.MapClaims{
		"sub":   userID,
		"email": email,
		"exp":   time.Now().Add(7 * 24 * time.Hour).Unix(),
		"iat":   time.Now().Unix(),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
}
