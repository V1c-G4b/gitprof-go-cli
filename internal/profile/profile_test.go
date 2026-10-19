package profile

import (
	"errors"
	"os"
	"path/filepath"
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

func TestExpandPath(t *testing.T) {
	home := filepath.FromSlash("/home/teste")
	tests := []struct {
		nome    string
		in      string
		want    string
		wantErr error
	}{
		{"til sozinho", "~", "/home/teste", nil},
		{"til com barra", "~/dev/work", "/home/teste/dev/work", nil},
		{"barra no fim", "~/dev/work/", "/home/teste/dev/work", nil},
		{"absoluto sujo", "/a/./b/../work//", "/a/work", nil},
		{"espacos", "  /a/work ", "/a/work", nil},
		{"vazio", "", "", ErrInvalidDir},
		{"til de outro usuario", "~outro/dev", "", ErrInvalidDir},
	}
	for _, tt := range tests {
		t.Run(tt.nome, func(t *testing.T) {
			got, err := ExpandPath(tt.in, home)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("got err %v, want %v", err, tt.wantErr)
			}
			if got != filepath.FromSlash(tt.want) {
				t.Fatalf("ExpandPath(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestExpandPathRelativo(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	got, err := ExpandPath("sub", "/home/teste")
	if err != nil {
		t.Fatal(err)
	}
	if got != filepath.Join(wd, "sub") {
		t.Fatalf("got %q, want %q", got, filepath.Join(wd, "sub"))
	}
}

func TestSetDir(t *testing.T) {
	tests := []struct {
		nome    string
		dir     string
		perfil  string
		wantErr error
	}{
		{"ok", "/a/work", "trabalho", nil},
		{"ignora maiusculas no nome", "/a/work", "TRABALHO", nil},
		{"vazio", "", "trabalho", ErrInvalidDir},
		{"relativo", "a/work", "trabalho", ErrInvalidDir},
		{"perfil inexistente", "/a/work", "freela", ErrProfileNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.nome, func(t *testing.T) {
			lista := []Profile{{Name: "trabalho", Email: "trab@exemplo.com"}}
			got, err := SetDir(filepath.FromSlash(tt.dir), lista, tt.perfil)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("got err %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr == nil && got[0].Dir != filepath.FromSlash(tt.dir) {
				t.Fatalf("got dir %q, want %q", got[0].Dir, tt.dir)
			}
		})
	}
}

func TestIsWithin(t *testing.T) {
	tests := []struct {
		nome string
		path string
		dir  string
		want bool
	}{
		{"igual", "/a/work", "/a/work", true},
		{"filho", "/a/work/repo", "/a/work", true},
		{"neto", "/a/work/x/repo", "/a/work", true},
		{"dir com barra no fim", "/a/work/repo", "/a/work/", true},
		{"prefixo sem separador", "/a/work2/repo", "/a/work", false},
		{"pai", "/a", "/a/work", false},
		{"irmao", "/a/pessoal/repo", "/a/work", false},
		{"raiz", "/a/repo", "/", true},
		{"dir vazio", "/a/work", "", false},
		{"path vazio", "", "/a/work", false},
	}
	for _, tt := range tests {
		t.Run(tt.nome, func(t *testing.T) {
			got := IsWithin(filepath.FromSlash(tt.path), filepath.FromSlash(tt.dir))
			if got != tt.want {
				t.Fatalf("IsWithin(%q, %q) = %v, want %v", tt.path, tt.dir, got, tt.want)
			}
		})
	}
}

func TestFindProfileByDir(t *testing.T) {
	lista := []Profile{
		{Name: "pessoal", Email: "teste@exemplo.com", Dir: filepath.FromSlash("/a")},
		{Name: "trabalho", Email: "trab@exemplo.com", Dir: filepath.FromSlash("/a/work")},
		{Name: "freela", Email: "freela@exemplo.com", Dir: filepath.FromSlash("/a/work/freela")},
		{Name: "semdir", Email: "semdir@exemplo.com"},
	}
	tests := []struct {
		nome    string
		path    string
		want    string
		wantErr error
	}{
		{"pasta do trabalho", "/a/work/repo", "trabalho", nil},
		{"prefixo mais longo vence", "/a/work/freela/repo", "freela", nil},
		{"work2 nao casa com work", "/a/work2/repo", "pessoal", nil},
		{"so o mais curto casa", "/a/outro", "pessoal", nil},
		{"fora de tudo", "/b/repo", "", ErrProfileNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.nome, func(t *testing.T) {
			got, err := FindProfileByDir(filepath.FromSlash(tt.path), lista)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("got err %v, want %v", err, tt.wantErr)
			}
			if got.Name != tt.want {
				t.Fatalf("got perfil %q, want %q", got.Name, tt.want)
			}
		})
	}
}

func TestFindProfileByDirOrdemNaoImporta(t *testing.T) {
	lista := []Profile{
		{Name: "trabalho", Email: "trab@exemplo.com", Dir: filepath.FromSlash("/a/work")},
		{Name: "pessoal", Email: "teste@exemplo.com", Dir: filepath.FromSlash("/a")},
	}
	got, err := FindProfileByDir(filepath.FromSlash("/a/work/repo"), lista)
	if err != nil || got.Name != "trabalho" {
		t.Fatalf("got (%q, %v), want trabalho", got.Name, err)
	}
}
