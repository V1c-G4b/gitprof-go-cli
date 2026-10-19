package profile

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

var (
	ErrEmptyName       = errors.New("name is empty")
	ErrEmptyEmail      = errors.New("email is empty")
	ErrInvalidEmail    = errors.New("invalid email")
	ErrNameExists      = errors.New("profile name already exists")
	ErrEmailExists     = errors.New("profile email already exists")
	ErrInvalidSSHHost  = errors.New("invalid ssh host")
	ErrProfileNotFound = errors.New("profile not found")
	ErrInvalidDir      = errors.New("invalid dir")
	ErrInvalidKey      = errors.New("invalid key")
	ErrInvalidUserName = errors.New("invalid user name")
)

type Profile struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	SSHHost  string `json:"ssh_host,omitempty"`
	Dir      string `json:"dir,omitempty"`
	Key      string `json:"key,omitempty"`
	UserName string `json:"user_name,omitempty"`
}

func New(name, email string) (Profile, error) {
	p := Profile{Name: strings.TrimSpace(name), Email: strings.TrimSpace(email)}
	if err := p.Validate(); err != nil {
		return Profile{}, err
	}
	return p, nil
}

func (p Profile) Validate() error {
	if p.Name == "" {
		return ErrEmptyName
	}
	if p.Email == "" {
		return ErrEmptyEmail
	}
	if !strings.Contains(p.Email, "@") {
		return ErrInvalidEmail
	}
	return nil
}

func Add(profile Profile, pl []Profile) ([]Profile, error) {
	for _, p := range pl {
		if strings.EqualFold(p.Name, profile.Name) {
			return nil, ErrNameExists
		}
		if strings.EqualFold(p.Email, profile.Email) {
			return nil, ErrEmailExists
		}
	}
	return append(pl, profile), nil
}

func SetSSHHost(host string, pl []Profile, name string) ([]Profile, error) {
	if strings.TrimSpace(host) == "" {
		return nil, ErrInvalidSSHHost
	}
	for i := range pl {
		if strings.EqualFold(pl[i].Name, name) {
			pl[i].SSHHost = strings.TrimSpace(host)
			return pl, nil
		}
	}
	return nil, ErrProfileNotFound
}

func FindProfileByName(name string, pl []Profile) (Profile, error) {
	for _, p := range pl {
		if strings.EqualFold(p.Name, name) {
			return p, nil
		}
	}
	return Profile{}, ErrProfileNotFound
}

func FindProfileBySSHHost(host string, pl []Profile) (Profile, error) {
	for _, p := range pl {
		if strings.EqualFold(p.SSHHost, host) {
			return p, nil
		}
	}
	return Profile{}, ErrProfileNotFound
}

func ExpandPath(raw, home string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", ErrInvalidDir
	}
	if raw == "~" {
		raw = home
	} else if rest, ok := strings.CutPrefix(raw, "~/"); ok {
		raw = filepath.Join(home, rest)
	} else if strings.HasPrefix(raw, "~") {
		return "", ErrInvalidDir
	}
	abs, err := filepath.Abs(raw)
	if err != nil {
		return "", err
	}
	return filepath.Clean(abs), nil
}

func SetDir(dir string, pl []Profile, name string) ([]Profile, error) {
	if strings.TrimSpace(dir) == "" || !filepath.IsAbs(dir) {
		return nil, ErrInvalidDir
	}
	for i := range pl {
		if strings.EqualFold(pl[i].Name, name) {
			pl[i].Dir = filepath.Clean(dir)
			return pl, nil
		}
	}
	return nil, ErrProfileNotFound
}

func IsWithin(path, dir string) bool {
	if path == "" || dir == "" {
		return false
	}
	path = filepath.Clean(path)
	dir = filepath.Clean(dir)
	if path == dir {
		return true
	}
	if !strings.HasSuffix(dir, string(os.PathSeparator)) {
		dir += string(os.PathSeparator)
	}
	return strings.HasPrefix(path, dir)
}

func FindProfileByDir(path string, pl []Profile) (Profile, error) {
	best := -1
	for i, p := range pl {
		if !IsWithin(path, p.Dir) {
			continue
		}
		if best < 0 || len(filepath.Clean(p.Dir)) > len(filepath.Clean(pl[best].Dir)) {
			best = i
		}
	}
	if best < 0 {
		return Profile{}, ErrProfileNotFound
	}
	return pl[best], nil
}

func SetKey(key string, pl []Profile, name string) ([]Profile, error) {
	if strings.TrimSpace(key) == "" || !filepath.IsAbs(key) {
		return nil, ErrInvalidKey
	}
	for i := range pl {
		if strings.EqualFold(pl[i].Name, name) {
			pl[i].Key = filepath.Clean(key)
			return pl, nil
		}
	}
	return nil, ErrProfileNotFound
}

func SetUserName(userName string, pl []Profile, name string) ([]Profile, error) {
	userName = strings.TrimSpace(userName)
	if userName == "" || strings.ContainsAny(userName, "\n<>") {
		return nil, ErrInvalidUserName
	}
	for i := range pl {
		if strings.EqualFold(pl[i].Name, name) {
			pl[i].UserName = userName
			return pl, nil
		}
	}
	return nil, ErrProfileNotFound
}
