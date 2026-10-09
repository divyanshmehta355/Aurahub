package api

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/divyanshmehta355/aurahub/backend/db"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/crypto/bcrypt"
)

type RegisterRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Avatar   string `json:"avatar"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// formatUUID converts a pgtype.UUID to a standard string representation
func formatUUID(id pgtype.UUID) string {
	b := id.Bytes
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func (s *Server) RegisterHandler(c *fiber.Ctx) error {
	var req RegisterRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid request payload"})
	}

	req.Username = strings.TrimSpace(req.Username)
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	if req.Username == "" || req.Email == "" || req.Password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "All fields are required."})
	}

	emailExists, err := s.Repository.CheckEmailExists(context.Background(), req.Email)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Failed to check email"})
	}
	if emailExists {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"message": "User with this email already exists."})
	}
	usernameExists, err := s.Repository.CheckUsernameExists(context.Background(), req.Username)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Failed to check username"})
	}
	if usernameExists {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"message": "Username is already taken."})
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), 10)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Error hashing password"})
	}

	avatar := req.Avatar
	if avatar == "" {
		avatar = fmt.Sprintf("/api/avatar/%s", url.QueryEscape(req.Username))
	}

	user, err := s.Repository.CreateUser(context.Background(), db.CreateUserParams{
		Email:    req.Email,
		Password: string(hashedPassword),
		Username: req.Username,
		Avatar:   pgtype.Text{String: avatar, Valid: true},
	})
	if err != nil {
		fmt.Printf("CreateUser Error: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Failed to create user"})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "User created successfully.",
		"user": fiber.Map{
			"id":       formatUUID(user.ID),
			"username": user.Username,
			"email":    user.Email,
		},
	})
}

func (s *Server) LoginHandler(c *fiber.Ctx) error {
	var req LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid request payload"})
	}

	user, err := s.Repository.GetUserByEmail(context.Background(), req.Email)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Invalid credentials"})
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password))
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Invalid credentials"})
	}

	// Generate JWT
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Authentication is not configured"})
	}

	userIdStr := formatUUID(user.ID)

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": userIdStr,
		"exp": time.Now().Add(time.Hour * 24 * 7).Unix(), // 7 days
	})

	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Could not login"})
	}

	c.Cookie(&fiber.Cookie{
		Name:     "jwt",
		Value:    tokenString,
		Path:     "/",
		Expires:  time.Now().Add(time.Hour * 24 * 7),
		HTTPOnly: true,
		Secure:   true,
		SameSite: "None",
	})

	return c.JSON(fiber.Map{
		"message": "success",
		"user": fiber.Map{
			"id":       userIdStr,
			"username": user.Username,
			"email":    user.Email,
			"avatar":   user.Avatar.String,
		},
	})
}

func (s *Server) LogoutHandler(c *fiber.Ctx) error {
	c.Cookie(&fiber.Cookie{
		Name:     "jwt",
		Value:    "",
		Path:     "/",
		Expires:  time.Now().Add(-time.Hour),
		HTTPOnly: true,
		Secure:   true,
		SameSite: "None",
	})
	return c.JSON(fiber.Map{"message": "Logged out successfully"})
}

func (s *Server) GetMeHandler(c *fiber.Ctx) error {
	userIdLocal := c.Locals("userId")
	if userIdLocal == nil {
		return c.Status(401).JSON(fiber.Map{"message": "Unauthorized"})
	}
	userId, err := parseUUID(userIdLocal.(string))
	if err != nil {
		return c.Status(401).JSON(fiber.Map{"message": "Unauthorized"})
	}

	user, err := s.Repository.GetUser(context.Background(), userId)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"message": "User not found"})
	}

	return c.JSON(fiber.Map{
		"id":               formatUUID(user.ID),
		"username":         user.Username,
		"email":            user.Email,
		"avatar":           user.Avatar.String,
		"banner":           user.Banner.String,
		"bio":              user.Bio.String,
		"showAdultContent": user.ShowAdultContent.Bool,
	})
}

func (s *Server) CheckEmailHandler(c *fiber.Ctx) error {
	var req struct {
		Email string `json:"email"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"available": false, "message": "Invalid request"})
	}

	if req.Email == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"available": false, "message": "Email is required."})
	}

	exists, err := s.Repository.CheckEmailExists(context.Background(), req.Email)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Server error"})
	}

	if exists {
		return c.JSON(fiber.Map{"available": false, "message": "An account with this email already exists."})
	}

	return c.JSON(fiber.Map{"available": true})
}

func (s *Server) CheckUsernameHandler(c *fiber.Ctx) error {
	var req struct {
		Username string `json:"username"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"available": false, "message": "Invalid request"})
	}

	if req.Username == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"available": false, "message": "Username is required."})
	}

	exists, err := s.Repository.CheckUsernameExists(context.Background(), req.Username)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Server error"})
	}

	if exists {
		return c.JSON(fiber.Map{"available": false, "message": "Username is already taken."})
	}

	return c.JSON(fiber.Map{"available": true})
}
