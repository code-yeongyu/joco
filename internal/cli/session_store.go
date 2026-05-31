package cli

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/yeongyu/jocohunt/internal/jocohunt"
)

func loadStoredSession(authFile string) (jocohunt.AuthSession, error) {
	path, err := sessionPath(authFile)
	if err != nil {
		return jocohunt.AuthSession{}, err
	}
	session, err := jocohunt.LoadAuthSession(path)
	if err != nil {
		if errors.Is(err, jocohunt.ErrNoAuthSession) {
			return jocohunt.AuthSession{}, nil
		}
		return jocohunt.AuthSession{}, err
	}
	return session, nil
}

func deleteStoredSession(authFile string) (string, error) {
	path, err := sessionPath(authFile)
	if err != nil {
		return "", err
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("remove session: %w", err)
	}
	return path, nil
}

func sessionPath(authFile string) (string, error) {
	if strings.TrimSpace(authFile) != "" {
		return authFile, nil
	}
	path, err := jocohunt.ResolveAuthFilePath("")
	if err != nil {
		return "", fmt.Errorf("find config dir: %w", err)
	}
	return path, nil
}
