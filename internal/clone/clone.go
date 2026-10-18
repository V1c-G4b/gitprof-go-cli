package clone

import (
	"errors"
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

func ExecClone(rawURL string) error {
	cmd := exec.Command("git", "clone", rawURL)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}
