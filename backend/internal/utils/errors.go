package utils

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
)

// AppError représente une erreur applicative
type AppError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

func (e *AppError) Error() string {
	return e.Message
}

// Constructeurs d'erreurs communes
func NewBadRequestError(message string) *AppError {
	return &AppError{
		Code:    fiber.StatusBadRequest,
		Message: message,
	}
}

func NewNotFoundError(resource string) *AppError {
	return &AppError{
		Code:    fiber.StatusNotFound,
		Message: fmt.Sprintf("%s not found", resource),
	}
}

func NewInternalError(message string) *AppError {
	return &AppError{
		Code:    fiber.StatusInternalServerError,
		Message: message,
	}
}

func NewUnauthorizedError(message string) *AppError {
	return &AppError{
		Code:    fiber.StatusUnauthorized,
		Message: message,
	}
}

func NewForbiddenError(message string) *AppError {
	return &AppError{
		Code:    fiber.StatusForbidden,
		Message: message,
	}
}

func NewRateLimitError(message string) *AppError {
	return &AppError{
		Code:    fiber.StatusTooManyRequests,
		Message: message,
	}
}

// ErrorResponse format de réponse d'erreur standardisé
type ErrorResponse struct {
	Error string `json:"error"`
	Code  int    `json:"code"`
}

// SendError envoie une erreur formatée au client
func SendError(c *fiber.Ctx, err error) error {
	if appErr, ok := err.(*AppError); ok {
		return c.Status(appErr.Code).JSON(ErrorResponse{
			Error: appErr.Message,
			Code:  appErr.Code,
		})
	}

	// Erreur non typée = 500
	return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
		Error: "Internal server error",
		Code:  fiber.StatusInternalServerError,
	})
}
