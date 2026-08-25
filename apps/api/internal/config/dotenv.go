package config

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// LoadDotEnv reads KEY=VALUE pairs from the first .env file it finds, walking up
// from the working directory toward the repo root. Existing environment
// variables always win, so a real deployment is never overridden by a stray
// local file.
//
// Vercel injects env vars directly and Next.js loads .env.local on its own, but
// a plain `go run ./cmd/api` gets neither — this closes that gap without taking
// on a dependency.
func LoadDotEnv(names ...string) {
	if len(names) == 0 {
		names = []string{".env.local", ".env"}
	}
	dir, err := os.Getwd()
	if err != nil {
		return
	}
	for i := 0; i < 5; i++ {
		for _, name := range names {
			if loadFile(filepath.Join(dir, name)) {
				return
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return
		}
		dir = parent
	}
}

func loadFile(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		key, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if len(value) >= 2 {
			if (value[0] == '"' && value[len(value)-1] == '"') ||
				(value[0] == '\'' && value[len(value)-1] == '\'') {
				value = value[1 : len(value)-1]
			}
		}
		if _, exists := os.LookupEnv(key); !exists {
			os.Setenv(key, value)
		}
	}
	return true
}
