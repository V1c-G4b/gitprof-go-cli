package profile

import (
	"errors"
	"strings"
)

var (
	ErrEmptyName    = errors.New("name is empty")
	ErrEmptyEmail   = errors.New("email is empty")
	ErrInvalidEmail = errors.New("invalid email")
	ErrProfileExist = errors.New("profile already exists")
)

type Profile struct {
	Name  string `json:"name"`
	Email string `json:"email"`
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
		if strings.ToUpper(p.Email) == strings.ToUpper(profile.Email) || strings.ToUpper(p.Name) == strings.ToUpper(profile.Name) {
			return nil, ErrProfileExist
		}
	}
	return append(pl, profile), nil
}
