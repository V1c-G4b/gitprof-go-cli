package clone

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

var (
	ErrInvalidURL = errors.New("invalid url")
)

func RewriteURL(rawURL string, alias string) (string, error) {
	if path, ok := strings.CutPrefix(rawURL, "git@github.com:"); ok {
		return "git@" + alias + ":" + path, nil
	}
	if path, ok := strings.CutPrefix(rawURL, "https://github.com/"); ok {
		return "git@" + alias + ":" + path, nil
	}
	if path, ok := strings.CutPrefix(rawURL, "http://github.com/"); ok {
		return "git@" + alias + ":" + path, nil
	}
	return "", ErrInvalidURL
}

func ExecClone(rawURL string, dir string) error {
	cmd := exec.Command("git", "clone", rawURL, dir)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

func SetIdentity(dir, email string) error {
	cmd := exec.Command("git", "-C", dir, "config", "user.email", email)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git config %v: %s", err, out)
	}
	return nil
}

func RepoName(rawURL string) (string, error) {
	sanitizeURL := strings.TrimSuffix(rawURL, "/")
	i := strings.LastIndexAny(sanitizeURL, "/:")

	nome := sanitizeURL[i+1:]
	nome = strings.TrimSuffix(nome, ".git")
	if nome == "" {
		return "", ErrInvalidURL
	}
	return nome, nil
}
