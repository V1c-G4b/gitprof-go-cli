package doctor

import (
	"errors"
	"fmt"
	"net/url"
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

// HostFromRemote devolve o host (ou alias SSH) de uma URL de remote SSH.
// Aceita o formato scp ([user@]host:path) e ssh://[user@]host[:porta]/path.
// Devolve false para HTTPS, caminho local ou URL sem host: nesses casos o
// doctor não sabe dizer qual é o perfil.
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

	// Formato scp. Como no git, uma "/" antes do ":" indica caminho local.
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

// CheckEmail compara o email do repo com o esperado pelo perfil, sem
// diferenciar maiúsculas. Se não bater, devolve um erro que embrulha
// ErrEmailMismatch e mostra o perfil e os dois emails.
func CheckEmail(profileName, expected, current string) error {
	if strings.EqualFold(strings.TrimSpace(expected), strings.TrimSpace(current)) {
		return nil
	}
	return fmt.Errorf("perfil %q espera %s, mas o repo usa %s: %w", profileName, expected, current, ErrEmailMismatch)
}
