package doctor

import (
	"errors"
	"strings"
	"testing"
)

func TestHostFromRemote(t *testing.T) {
	tests := []struct {
		nome   string
		in     string
		want   string
		wantOK bool
	}{
		{"scp padrao", "git@github.com:o/r.git", "github.com", true},
		{"scp com alias", "git@github-trab:o/r.git", "github-trab", true},
		{"scp com user diferente de git", "deploy@servidor:o/r.git", "servidor", true},
		{"scp sem user", "github.com:o/r.git", "github.com", true},
		{"scp com espacos", "  git@github.com:o/r.git\n", "github.com", true},
		{"ssh url", "ssh://git@github.com/o/r.git", "github.com", true},
		{"ssh url com porta", "ssh://git@github.com:22/o/r.git", "github.com", true},
		{"ssh url com alias", "ssh://git@github-trab/o/r.git", "github-trab", true},
		{"git+ssh url", "git+ssh://git@github.com/o/r.git", "github.com", true},
		{"https", "https://github.com/o/r.git", "", false},
		{"http", "http://github.com/o/r.git", "", false},
		{"file url", "file:///tmp/r.git", "", false},
		{"caminho local", "/tmp/r.git", "", false},
		{"caminho relativo com dois pontos", "./dir:x/r.git", "", false},
		{"sem dois pontos", "git@github.com", "", false},
		{"sem path", "git@github.com:", "", false},
		{"sem host", "git@:o/r.git", "", false},
		{"vazio", "", "", false},
		{"ssh url sem host", "ssh:///o/r.git", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.nome, func(t *testing.T) {
			got, ok := HostFromRemote(tt.in)
			if got != tt.want || ok != tt.wantOK {
				t.Fatalf("HostFromRemote(%q) = (%q, %v), want (%q, %v)", tt.in, got, ok, tt.want, tt.wantOK)
			}
		})
	}
}

func TestCheckEmail(t *testing.T) {
	tests := []struct {
		nome     string
		expected string
		current  string
		wantErr  error
	}{
		{"igual", "teste@exemplo.com", "teste@exemplo.com", nil},
		{"ignora maiusculas", "teste@exemplo.com", "TESTE@Exemplo.com", nil},
		{"ignora espacos", " teste@exemplo.com ", "teste@exemplo.com", nil},
		{"diferente", "trab@exemplo.com", "teste@exemplo.com", ErrEmailMismatch},
	}
	for _, tt := range tests {
		t.Run(tt.nome, func(t *testing.T) {
			err := CheckEmail("trabalho", tt.expected, tt.current)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("got err %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr != nil {
				msg := err.Error()
				for _, parte := range []string{`"trabalho"`, tt.expected, tt.current} {
					if !strings.Contains(msg, parte) {
						t.Fatalf("mensagem %q sem %q", msg, parte)
					}
				}
			}
		})
	}
}
