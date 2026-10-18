package profile

import (
	"errors"
	"testing"
)

func TestNew(t *testing.T) {
	tests := []struct {
		nome    string
		name    string
		email   string
		wantErr error
	}{
		{"valido", "pessoal", "teste@exemplo.com", nil},
		{"tira espacos", "  pessoal ", " teste@exemplo.com ", nil},
		{"nome vazio", "", "teste@exemplo.com", ErrEmptyName},
		{"nome so com espacos", "   ", "teste@exemplo.com", ErrEmptyName},
		{"email vazio", "pessoal", "", ErrEmptyEmail},
		{"email sem arroba", "pessoal", "teste.exemplo.com", ErrInvalidEmail},
	}
	for _, tt := range tests {
		t.Run(tt.nome, func(t *testing.T) {
			p, err := New(tt.name, tt.email)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("got err %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr == nil && (p.Name != "pessoal" || p.Email != "teste@exemplo.com") {
				t.Fatalf("got %+v, want nome e email sem espacos", p)
			}
		})
	}
}

func TestAdd(t *testing.T) {
	existente := []Profile{{Name: "pessoal", Email: "teste@exemplo.com"}}

	tests := []struct {
		nome    string
		lista   []Profile
		novo    Profile
		wantErr error
		wantLen int
	}{
		{"lista vazia", nil, Profile{Name: "trab", Email: "trab@exemplo.com"}, nil, 1},
		{"sem repetido", existente, Profile{Name: "trab", Email: "trab@exemplo.com"}, nil, 2},
		{"nome repetido", existente, Profile{Name: "pessoal", Email: "outro@exemplo.com"}, ErrNameExists, 0},
		{"nome repetido ignorando maiusculas", existente, Profile{Name: "PESSOAL", Email: "outro@exemplo.com"}, ErrNameExists, 0},
		{"email repetido", existente, Profile{Name: "trab", Email: "teste@exemplo.com"}, ErrEmailExists, 0},
		{"email repetido ignorando maiusculas", existente, Profile{Name: "trab", Email: "TESTE@Exemplo.com"}, ErrEmailExists, 0},
	}
	for _, tt := range tests {
		t.Run(tt.nome, func(t *testing.T) {
			got, err := Add(tt.novo, tt.lista)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("got err %v, want %v", err, tt.wantErr)
			}
			if len(got) != tt.wantLen {
				t.Fatalf("got %d perfis, want %d", len(got), tt.wantLen)
			}
		})
	}
}
