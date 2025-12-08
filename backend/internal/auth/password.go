package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidPassword     = errors.New("invalid password")
	ErrPasswordTooShort    = errors.New("password must be at least 8 characters")
	ErrPasswordTooLong     = errors.New("password must be less than 72 characters")
	ErrPasswordTooWeak     = errors.New("password is too weak")
	ErrInvalidPasswordHash = errors.New("invalid password hash format")
)

const (
	// BcryptCost is the cost factor for bcrypt hashing
	// Higher is more secure but slower (4-31, default 10)
	BcryptCost = 12

	// MinPasswordLength is the minimum password length
	MinPasswordLength = 8

	// MaxPasswordLength is the maximum password length (bcrypt limit is 72)
	MaxPasswordLength = 72
)

// HashPassword hashes a password using bcrypt
func HashPassword(password string) (string, error) {
	// Validate password length
	if len(password) < MinPasswordLength {
		return "", ErrPasswordTooShort
	}
	if len(password) > MaxPasswordLength {
		return "", ErrPasswordTooLong
	}

	// Hash password
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), BcryptCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}

	return string(hashedBytes), nil
}

// ComparePassword compares a plaintext password with a hashed password
func ComparePassword(hashedPassword, password string) error {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	if err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return ErrInvalidPassword
		}
		return fmt.Errorf("failed to compare password: %w", err)
	}

	return nil
}

// ValidatePassword validates password strength
func ValidatePassword(password string) error {
	// Check length
	if len(password) < MinPasswordLength {
		return ErrPasswordTooShort
	}
	if len(password) > MaxPasswordLength {
		return ErrPasswordTooLong
	}

	// Check for at least one uppercase letter
	hasUpper := false
	hasLower := false
	hasDigit := false
	hasSpecial := false

	for _, char := range password {
		switch {
		case 'A' <= char && char <= 'Z':
			hasUpper = true
		case 'a' <= char && char <= 'z':
			hasLower = true
		case '0' <= char && char <= '9':
			hasDigit = true
		case isSpecialChar(char):
			hasSpecial = true
		}
	}

	// Require at least 3 of the 4 categories
	categories := 0
	if hasUpper {
		categories++
	}
	if hasLower {
		categories++
	}
	if hasDigit {
		categories++
	}
	if hasSpecial {
		categories++
	}

	if categories < 3 {
		return ErrPasswordTooWeak
	}

	// Check for common weak passwords
	if isCommonPassword(password) {
		return ErrPasswordTooWeak
	}

	return nil
}

// isSpecialChar checks if a character is a special character
func isSpecialChar(char rune) bool {
	specialChars := "!@#$%^&*()_+-=[]{}|;:,.<>?/"
	for _, special := range specialChars {
		if char == special {
			return true
		}
	}
	return false
}

// isCommonPassword checks if password is in common password list
func isCommonPassword(password string) bool {
	// List of common weak passwords
	commonPasswords := []string{
		"password", "12345678", "123456789", "qwerty", "abc123",
		"password1", "password123", "admin", "letmein", "welcome",
		"monkey", "dragon", "master", "sunshine", "princess",
		"football", "baseball", "shadow", "michael", "jennifer",
	}

	lowerPassword := strings.ToLower(password)
	for _, common := range commonPasswords {
		if lowerPassword == common {
			return true
		}
	}

	return false
}

// GenerateRandomPassword generates a random secure password
func GenerateRandomPassword(length int) (string, error) {
	if length < MinPasswordLength {
		length = MinPasswordLength
	}
	if length > MaxPasswordLength {
		length = MaxPasswordLength
	}

	// Character sets
	uppercase := "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	lowercase := "abcdefghijklmnopqrstuvwxyz"
	digits := "0123456789"
	special := "!@#$%^&*()_+-=[]{}|;:,.<>?/"
	allChars := uppercase + lowercase + digits + special

	// Ensure at least one character from each set
	password := make([]byte, length)

	// Add one character from each required set
	password[0] = uppercase[randomInt(len(uppercase))]
	password[1] = lowercase[randomInt(len(lowercase))]
	password[2] = digits[randomInt(len(digits))]
	password[3] = special[randomInt(len(special))]

	// Fill rest with random characters from all sets
	for i := 4; i < length; i++ {
		password[i] = allChars[randomInt(len(allChars))]
	}

	// Shuffle password
	for i := range password {
		j := randomInt(len(password))
		password[i], password[j] = password[j], password[i]
	}

	return string(password), nil
}

// randomInt generates a cryptographically secure random integer [0, max)
func randomInt(max int) int {
	if max <= 0 {
		return 0
	}

	// Calculate number of bytes needed
	bytes := make([]byte, 1)
	_, err := rand.Read(bytes)
	if err != nil {
		// Fallback to a deterministic but not secure method
		return 0
	}

	// Convert to int and modulo
	return int(bytes[0]) % max
}

// GenerateResetToken generates a secure password reset token
func GenerateResetToken() (string, error) {
	bytes := make([]byte, 32)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", fmt.Errorf("failed to generate reset token: %w", err)
	}

	return base64.URLEncoding.EncodeToString(bytes), nil
}

// HashResetToken hashes a reset token for storage
func HashResetToken(token string) (string, error) {
	// Use bcrypt with lower cost for tokens (they're already random)
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(token), bcrypt.MinCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash reset token: %w", err)
	}

	return string(hashedBytes), nil
}

// CompareResetToken compares a reset token with its hash
func CompareResetToken(hashedToken, token string) error {
	return ComparePassword(hashedToken, token)
}

// PasswordStrength calculates password strength score (0-5)
func PasswordStrength(password string) int {
	score := 0

	// Length
	if len(password) >= 8 {
		score++
	}
	if len(password) >= 12 {
		score++
	}
	if len(password) >= 16 {
		score++
	}

	// Character variety
	hasUpper := false
	hasLower := false
	hasDigit := false
	hasSpecial := false

	for _, char := range password {
		switch {
		case 'A' <= char && char <= 'Z':
			hasUpper = true
		case 'a' <= char && char <= 'z':
			hasLower = true
		case '0' <= char && char <= '9':
			hasDigit = true
		case isSpecialChar(char):
			hasSpecial = true
		}
	}

	if hasUpper && hasLower {
		score++
	}
	if hasDigit {
		score++
	}
	if hasSpecial {
		score++
	}

	// Penalize common passwords
	if isCommonPassword(password) {
		score = 0
	}

	// Cap at 5
	if score > 5 {
		score = 5
	}

	return score
}

// SecureCompare performs constant-time comparison of two strings
// Use this to prevent timing attacks when comparing secrets
func SecureCompare(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

// PasswordPolicy represents password requirements
type PasswordPolicy struct {
	MinLength      int
	MaxLength      int
	RequireUpper   bool
	RequireLower   bool
	RequireDigit   bool
	RequireSpecial bool
	MinCategories  int // Minimum number of character categories required
}

// DefaultPasswordPolicy returns the default password policy
func DefaultPasswordPolicy() PasswordPolicy {
	return PasswordPolicy{
		MinLength:      MinPasswordLength,
		MaxLength:      MaxPasswordLength,
		RequireUpper:   false,
		RequireLower:   false,
		RequireDigit:   false,
		RequireSpecial: false,
		MinCategories:  3, // Require at least 3 of: upper, lower, digit, special
	}
}

// StrictPasswordPolicy returns a strict password policy
func StrictPasswordPolicy() PasswordPolicy {
	return PasswordPolicy{
		MinLength:      12,
		MaxLength:      MaxPasswordLength,
		RequireUpper:   true,
		RequireLower:   true,
		RequireDigit:   true,
		RequireSpecial: true,
		MinCategories:  4,
	}
}

// ValidateWithPolicy validates a password against a policy
func ValidateWithPolicy(password string, policy PasswordPolicy) error {
	// Check length
	if len(password) < policy.MinLength {
		return fmt.Errorf("password must be at least %d characters", policy.MinLength)
	}
	if len(password) > policy.MaxLength {
		return fmt.Errorf("password must be less than %d characters", policy.MaxLength)
	}

	// Check character requirements
	hasUpper := false
	hasLower := false
	hasDigit := false
	hasSpecial := false

	for _, char := range password {
		switch {
		case 'A' <= char && char <= 'Z':
			hasUpper = true
		case 'a' <= char && char <= 'z':
			hasLower = true
		case '0' <= char && char <= '9':
			hasDigit = true
		case isSpecialChar(char):
			hasSpecial = true
		}
	}

	// Check specific requirements
	if policy.RequireUpper && !hasUpper {
		return errors.New("password must contain at least one uppercase letter")
	}
	if policy.RequireLower && !hasLower {
		return errors.New("password must contain at least one lowercase letter")
	}
	if policy.RequireDigit && !hasDigit {
		return errors.New("password must contain at least one digit")
	}
	if policy.RequireSpecial && !hasSpecial {
		return errors.New("password must contain at least one special character")
	}

	// Check minimum categories
	categories := 0
	if hasUpper {
		categories++
	}
	if hasLower {
		categories++
	}
	if hasDigit {
		categories++
	}
	if hasSpecial {
		categories++
	}

	if categories < policy.MinCategories {
		return fmt.Errorf("password must contain at least %d different character types", policy.MinCategories)
	}

	// Check for common weak passwords
	if isCommonPassword(password) {
		return errors.New("password is too common")
	}

	return nil
}
