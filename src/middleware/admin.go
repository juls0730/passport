package middleware

import (
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v3"
)

type Session struct {
	SessionID string `json:"session_id"`
	ExpiresAt string `json:"expires_at"`
}

func AdminMiddleware(db *sql.DB) func(c fiber.Ctx) error {
	return func(c fiber.Ctx) error {
		sessionToken := c.Cookies("SessionToken")
		if sessionToken == "" {
			return c.Next()
		}

		// Check if session exists
		var session Session
		err := db.QueryRow(`
			SELECT session_id, expires_at 
			FROM sessions 
			WHERE session_id = ?
		`, sessionToken).Scan(&session.SessionID, &session.ExpiresAt)
		if err != nil {
			slog.Error("Failed to check session", "error", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"message": fmt.Sprintf("Failed to check session: %v", err),
			})
		}

		sessionExpiry, err := time.Parse("2006-01-02 15:04:05-07:00", session.ExpiresAt)
		if err != nil {
			slog.Error("Failed to parse session expiry", "error", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"message": fmt.Sprintf("Failed to parse session expiry: %v", err),
			})
		}

		if sessionExpiry.Before(time.Now()) {
			return c.Next()
		}

		c.Locals("IsAdmin", true)
		return c.Next()
	}
}
