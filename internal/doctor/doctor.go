package doctor

import (
	"errors"
	"os/exec"
	"strings"
)

var (
	ErrEmailMismatch = errors.New("email mismatch")
	ErrMissingRemote = errors.New("missing remote")
)

func CurrentEmail(dir string) (string, error) {
	out, err := exec.Command("git", "-C", dir, "config", "user.email").Output()
	if err != nil {
		var exitError *exec.ExitError
		if errors.As(err, &exitError) && exitError.ExitCode() == 1 {
			return "", nil
		}
		return "", err
	}

	return strings.TrimSpace(string(out)), nil
}

func ExpectedRemoteURL(dir string) (string, error) {
	out, err := exec.Command("git", "-C", dir, "remote", "get-url", "origin").Output()
	if err != nil {
		var exitError *exec.ExitError
		if errors.As(err, &exitError) && exitError.ExitCode() == 2 {
			return "", ErrMissingRemote
		}
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
