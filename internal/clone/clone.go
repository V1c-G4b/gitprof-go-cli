package clone

import (
	"errors"
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
