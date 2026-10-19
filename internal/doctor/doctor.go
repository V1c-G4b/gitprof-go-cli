package doctor

import (
	"errors"
	"fmt"
	"gitprof-go-cli/internal/profile"
	"net/url"
	"os/exec"
	"strings"
)

var (
	ErrEmailMismatch   = errors.New("email mismatch")
	ErrMissingRemote   = errors.New("missing remote")
	ErrProfileConflict = errors.New("profile conflict")
)

func IsMismatch(err error) bool {
	return errors.Is(err, ErrEmailMismatch) || errors.Is(err, ErrProfileConflict)
}

func Toplevel(dir string) (string, error) {
	out, err := exec.Command("git", "-C", dir, "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

type Match struct {
	Profile profile.Profile
	Found   bool
}

func Decide(byDir, byHost Match, host string) (profile.Profile, bool, error) {
	switch {
	case byDir.Found && byHost.Found:
		if strings.EqualFold(byDir.Profile.Name, byHost.Profile.Name) {
			return byDir.Profile, true, nil
		}
		return profile.Profile{}, false, fmt.Errorf(
			"a pasta do repo pertence ao perfil %q (%s), mas o origin usa o alias %q do perfil %q: %w",
			byDir.Profile.Name, byDir.Profile.Dir, host, byHost.Profile.Name, ErrProfileConflict)
	case byDir.Found:
		return byDir.Profile, true, nil
	case byHost.Found:
		return byHost.Profile, true, nil
	}
	return profile.Profile{}, false, nil
}

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

func HostFromRemote(rawURL string) (string, bool) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return "", false
	}

	if strings.Contains(rawURL, "://") {
		u, err := url.Parse(rawURL)
		if err != nil || (u.Scheme != "ssh" && u.Scheme != "git+ssh") {
			return "", false
		}
		host := u.Hostname()
		return host, host != ""
	}

	hostPart, path, ok := strings.Cut(rawURL, ":")
	if !ok || path == "" || strings.Contains(hostPart, "/") {
		return "", false
	}
	if i := strings.LastIndex(hostPart, "@"); i >= 0 {
		hostPart = hostPart[i+1:]
	}
	if hostPart == "" {
		return "", false
	}
	return hostPart, true
}

func CheckEmail(profileName, expected, current string) error {
	if strings.EqualFold(strings.TrimSpace(expected), strings.TrimSpace(current)) {
		return nil
	}
	return fmt.Errorf("perfil %q espera %s, mas o repo usa %s: %w", profileName, expected, current, ErrEmailMismatch)
}
