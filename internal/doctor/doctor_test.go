package doctor

import (
	"errors"
	"fmt"
	"gitprof-go-cli/internal/profile"
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

func TestDecide(t *testing.T) {
	pessoal := profile.Profile{Name: "pessoal", Email: "teste@exemplo.com", SSHHost: "github.com", Dir: "/a/pessoal"}
	trabalho := profile.Profile{Name: "trabalho", Email: "trab@exemplo.com", SSHHost: "github-trab", Dir: "/a/work"}
	trabalhoMaiusculo := trabalho
	trabalhoMaiusculo.Name = "Trabalho"
	nada := Match{}

	tests := []struct {
		nome    string
		byDir   Match
		byHost  Match
		want    string
		wantOK  bool
		wantErr error
	}{
		{"pasta e alias concordam", Match{trabalho, true}, Match{trabalho, true}, "trabalho", true, nil},
		{"concordam ignorando maiusculas", Match{trabalho, true}, Match{trabalhoMaiusculo, true}, "trabalho", true, nil},
		{"pasta e alias divergem", Match{trabalho, true}, Match{pessoal, true}, "", false, ErrProfileConflict},
		{"so a pasta", Match{trabalho, true}, nada, "trabalho", true, nil},
		{"so o alias", nada, Match{pessoal, true}, "pessoal", true, nil},
		{"nenhum", nada, nada, "", false, nil},
	}
	for _, tt := range tests {
		t.Run(tt.nome, func(t *testing.T) {
			got, ok, err := Decide(tt.byDir, tt.byHost, "github.com")
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("got err %v, want %v", err, tt.wantErr)
			}
			if got.Name != tt.want || ok != tt.wantOK {
				t.Fatalf("got (%q, %v), want (%q, %v)", got.Name, ok, tt.want, tt.wantOK)
			}
			if tt.wantErr != nil {
				msg := err.Error()
				for _, parte := range []string{`"trabalho"`, `"pessoal"`, `"github.com"`, "/a/work"} {
					if !strings.Contains(msg, parte) {
						t.Fatalf("mensagem %q sem %q", msg, parte)
					}
				}
			}
		})
	}
}

func TestIsMismatch(t *testing.T) {
	tests := []struct {
		nome string
		err  error
		want bool
	}{
		{"email", fmt.Errorf("x: %w", ErrEmailMismatch), true},
		{"conflito", fmt.Errorf("x: %w", ErrProfileConflict), true},
		{"remote", ErrMissingRemote, false},
		{"outro", errors.New("falha"), false},
		{"nil", nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.nome, func(t *testing.T) {
			if got := IsMismatch(tt.err); got != tt.want {
				t.Fatalf("IsMismatch(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}
