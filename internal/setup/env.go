package setup

import (
	"os"
	"path/filepath"
)

const (
	IncludeLine  = "Include ~/.ssh/config.d/gitprof"
	BackupSuffix = ".bak-gitprof"
)

type Env struct {
	Home string
}

func (e Env) SSHConfig() string {
	return filepath.Join(e.Home, ".ssh", "config")
}

func (e Env) SSHFragment() string {
	return filepath.Join(e.Home, ".ssh", "config.d", "gitprof")
}

func (e Env) ConfigDir() string {
	return filepath.Join(e.Home, ".config", "gitprof")
}

func (e Env) HooksDir() string {
	return filepath.Join(e.ConfigDir(), "hooks")
}

func (e Env) HookFile() string {
	return filepath.Join(e.HooksDir(), "pre-commit")
}

func (e Env) ProfileGitconfig(name string) string {
	return filepath.Join(e.ConfigDir(), name+".gitconfig")
}

func (e Env) GlobalGitconfig() string {
	if p := os.Getenv("GIT_CONFIG_GLOBAL"); p != "" {
		return p
	}
	home := filepath.Join(e.Home, ".gitconfig")
	if fileExists(home) {
		return home
	}
	xdg := os.Getenv("XDG_CONFIG_HOME")
	if xdg == "" {
		xdg = filepath.Join(e.Home, ".config")
	}
	if p := filepath.Join(xdg, "git", "config"); fileExists(p) {
		return p
	}
	return home
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}
