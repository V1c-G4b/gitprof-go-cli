package profile

import (
	"errors"
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
)

type Profile struct {
	Name    string `json:"name"`
	Email   string `json:"email"`
	SSHHost string `json:"ssh_host,omitempty"`
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
