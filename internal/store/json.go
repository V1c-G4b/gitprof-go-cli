package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"gitprof-go-cli/internal/profile"
	"io/fs"
	"os"
	"path/filepath"
)

func pathConfig() (string, error) {
	base, err := os.UserConfigDir()

	if err != nil {
		return "", fmt.Errorf("buscar diretório de config: %w", err)
	}

	return filepath.Join(base, "gitprof", "profiles.json"), nil
}

func Save(profiles []profile.Profile) (string, error) {
	path, err := pathConfig()
	if err != nil {
		return "", err
	}

	indent, err := json.MarshalIndent(profiles, "", "  ")
	if err != nil {
		return "", fmt.Errorf("gerar json: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return "", fmt.Errorf("criar pasta de config: %w", err)
	}

	if err := os.WriteFile(path, indent, 0600); err != nil {
		return "", fmt.Errorf("salvar %s: %w", path, err)
	}
	return path, nil
}

func Load() ([]profile.Profile, error) {
	path, err := pathConfig()

	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return []profile.Profile{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("ler %s: %w", path, err)
	}

	var lista []profile.Profile

	if err := json.Unmarshal(data, &lista); err != nil {
		return nil, fmt.Errorf("decodificar %s: %w", path, err)
	}

	return lista, nil

}
