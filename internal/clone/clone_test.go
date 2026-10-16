package clone

import (
	"errors"
	"testing"
)

func TestRewriteURL(t *testing.T) {
	tests := []struct {
		in      string
		alias   string
		want    string
		wantErr error
	}{
		{"git@github.com:org/repo.git", "github-trabalho", "git@github-trabalho:org/repo.git", nil},
		{"https://github.com/org/repo.git", "github-trabalho", "git@github-trabalho:org/repo.git", nil},
		{"http://github.com/org/repo.git", "github-trabalho", "git@github-trabalho:org/repo.git", nil},
		{"git@gitlab.com:org/repo.git", "github-trabalho", "", ErrInvalidURL},
		{"invalid-url", "github-trabalho", "", ErrInvalidURL},
	}
	for _, tt := range tests {
		got, err := RewriteURL(tt.in, tt.alias)
		if !errors.Is(err, tt.wantErr) || got != tt.want {
			t.Errorf("RewriteURL(%q, %q) = (%q, %v), want (%q, %v)", tt.in, tt.alias, got, err, tt.want, tt.wantErr)
		}
	}
}
