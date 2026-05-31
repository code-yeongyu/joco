package jocohunt

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const AuthFileEnv = "JOCOHUNT_AUTH_FILE"

var ErrNoAuthSession = errors.New("jocohunt auth: no saved session")

type AuthSession struct {
	BaseURL       string    `json:"baseURL"`
	SessionCookie string    `json:"sessionCookie"`
	CSRFToken     string    `json:"csrfToken,omitempty"`
	CreatedAt     time.Time `json:"createdAt"`
}

func ResolveAuthFilePath(override string) (string, error) {
	if path := strings.TrimSpace(override); path != "" {
		return path, nil
	}
	if path := strings.TrimSpace(os.Getenv(AuthFileEnv)); path != "" {
		return path, nil
	}
	if dir := strings.TrimSpace(os.Getenv("JOCOHUNT_CONFIG_DIR")); dir != "" {
		return filepath.Join(dir, "session.json"), nil
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("find config dir: %w", err)
	}
	return filepath.Join(dir, "jocohunt", "session.json"), nil
}

func SaveAuthSession(path string, session AuthSession) (err error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return errorsForAuth("auth file path is required")
	}
	session.BaseURL = strings.TrimSpace(session.BaseURL)
	session.SessionCookie = strings.TrimSpace(session.SessionCookie)
	session.CSRFToken = strings.TrimSpace(session.CSRFToken)
	if session.SessionCookie == "" {
		return errorsForAuth("session cookie is required")
	}
	if session.CreatedAt.IsZero() {
		session.CreatedAt = time.Now().UTC()
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create auth dir: %w", err)
	}
	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return fmt.Errorf("encode auth session: %w", err)
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("open auth session: %w", err)
	}
	defer func() {
		err = errors.Join(err, file.Close())
	}()
	if _, err := file.Write(data); err != nil {
		return fmt.Errorf("write auth session: %w", err)
	}
	if _, err := file.Write([]byte("\n")); err != nil {
		return fmt.Errorf("write auth session newline: %w", err)
	}
	if err := file.Chmod(0o600); err != nil {
		return fmt.Errorf("chmod auth session: %w", err)
	}
	return nil
}

func LoadAuthSession(path string) (AuthSession, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return AuthSession{}, errorsForAuth("auth file path is required")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return AuthSession{}, ErrNoAuthSession
		}
		return AuthSession{}, fmt.Errorf("read auth session: %w", err)
	}
	var session AuthSession
	if err := json.Unmarshal(data, &session); err != nil {
		return AuthSession{}, fmt.Errorf("decode auth file: %w", err)
	}
	session.BaseURL = strings.TrimSpace(session.BaseURL)
	session.SessionCookie = strings.TrimSpace(session.SessionCookie)
	session.CSRFToken = strings.TrimSpace(session.CSRFToken)
	if session.SessionCookie == "" {
		return AuthSession{}, errorsForAuth("saved session cookie is empty")
	}
	return session, nil
}

func DeleteAuthSession(path string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return errorsForAuth("auth file path is required")
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("delete auth session: %w", err)
	}
	return nil
}
