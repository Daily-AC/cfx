package extension

import (
	"crypto/rand"
	"embed"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/Daily-AC/cfx/internal/config"
)

//go:embed assets/*
var assets embed.FS

const tokenPlaceholder = "CFX_EXTENSION_TOKEN"

func Install(token string) (string, string, error) {
	dir, err := config.Dir()
	if err != nil {
		return "", "", err
	}
	if token == "" {
		token, err = newToken()
		if err != nil {
			return "", "", err
		}
	}
	target := filepath.Join(dir, "extension")
	if err := os.MkdirAll(target, 0o700); err != nil {
		return "", "", err
	}
	entries, err := fs.ReadDir(assets, "assets")
	if err != nil {
		return "", "", err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		body, err := assets.ReadFile(path.Join("assets", entry.Name()))
		if err != nil {
			return "", "", err
		}
		if entry.Name() == "service-worker.js" {
			body = []byte(strings.ReplaceAll(string(body), tokenPlaceholder, token))
		}
		if err := os.WriteFile(filepath.Join(target, entry.Name()), body, 0o600); err != nil {
			return "", "", err
		}
	}
	manifest := filepath.Join(target, "manifest.json")
	if _, err := os.Stat(manifest); err != nil {
		return "", "", fmt.Errorf("extension install is incomplete: %w", err)
	}
	return target, token, nil
}

func newToken() (string, error) {
	var value [32]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(value[:]), nil
}

func Path() (string, error) {
	dir, err := config.Dir()
	if err != nil {
		return "", err
	}
	path := filepath.Join(dir, "extension")
	if _, err := os.Stat(filepath.Join(path, "manifest.json")); err != nil {
		return "", fmt.Errorf("cfx extension is not installed; run `cfx extension install`: %w", err)
	}
	return path, nil
}
