package doctor

import (
	"errors"
	"os/exec"
	"strings"
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
