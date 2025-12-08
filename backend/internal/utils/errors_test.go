package utils

import (
	"testing"

	"github.com/gofiber/fiber/v2"
)

func TestAppError(t *testing.T) {
	err := NewBadRequestError("invalid input")

	if err.Code != fiber.StatusBadRequest {
		t.Errorf("Expected code 400, got %d", err.Code)
	}

	if err.Message != "invalid input" {
		t.Errorf("Expected message 'invalid input', got '%s'", err.Message)
	}
}

func TestErrorConstructors(t *testing.T) {
	tests := []struct {
		name     string
		err      *AppError
		wantCode int
	}{
		{"BadRequest", NewBadRequestError("test"), 400},
		{"NotFound", NewNotFoundError("user"), 404},
		{"Unauthorized", NewUnauthorizedError("test"), 401},
		{"Forbidden", NewForbiddenError("test"), 403},
		{"RateLimit", NewRateLimitError("test"), 429},
		{"Internal", NewInternalError("test"), 500},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Code != tt.wantCode {
				t.Errorf("Expected code %d, got %d", tt.wantCode, tt.err.Code)
			}
		})
	}
}
