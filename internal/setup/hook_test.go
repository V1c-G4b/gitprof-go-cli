package setup

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
)

func fakeDoctorDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	script := "#!/bin/sh\nexit \"$FAKE_RC\"\n"
	if err := os.WriteFile(filepath.Join(dir, "gitprof"), []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	return dir
}

func runSh(t *testing.T, dir, script string, env ...string) int {
	t.Helper()
	sh, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("sem sh")
	}
	cmd := exec.Command(sh, "-c", script)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), env...)
	err = cmd.Run()
	var exitError *exec.ExitError
	if errors.As(err, &exitError) {
		return exitError.ExitCode()
	}
	if err != nil {
		t.Fatal(err)
	}
	return 0
}

func TestHookSnippetNoFimDoScript(t *testing.T) {
	bin := fakeDoctorDir(t)
	home := t.TempDir()
	script := "true\n" + HookSnippet
	for _, tt := range []struct{ doctor, want int }{{0, 0}, {1, 1}, {2, 0}} {
		t.Run("doctor "+strconv.Itoa(tt.doctor), func(t *testing.T) {
			got := runSh(t, home, script, "PATH="+bin, "HOME="+home, "FAKE_RC="+strconv.Itoa(tt.doctor))
			if got != tt.want {
				t.Fatalf("status = %d, want %d", got, tt.want)
			}
		})
	}
	t.Run("sem binario", func(t *testing.T) {
		if got := runSh(t, home, script, "PATH="+t.TempDir(), "HOME="+home); got != 0 {
			t.Fatalf("status = %d, want 0", got)
		}
	})
}

func TestHookSnippetAntesDeOutroComando(t *testing.T) {
	bin := fakeDoctorDir(t)
	home := t.TempDir()
	script := HookSnippet + "exit 7\n"
	for _, tt := range []struct{ doctor, want int }{{0, 7}, {1, 1}, {2, 7}} {
		t.Run("doctor "+strconv.Itoa(tt.doctor), func(t *testing.T) {
			got := runSh(t, home, script, "PATH="+bin, "HOME="+home, "FAKE_RC="+strconv.Itoa(tt.doctor))
			if got != tt.want {
				t.Fatalf("status = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestHookScript(t *testing.T) {
	gitPath, err := exec.LookPath("git")
	if err != nil {
		t.Skip("sem git")
	}
	bin := fakeDoctorDir(t)
	home := t.TempDir()
	repo := t.TempDir()
	if out, err := exec.Command(gitPath, "init", "-q", repo).CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, out)
	}
	path := bin + string(os.PathListSeparator) + filepath.Dir(gitPath) + string(os.PathListSeparator) + os.Getenv("PATH")
	for _, tt := range []struct{ doctor, want int }{{0, 0}, {1, 1}, {2, 0}} {
		t.Run("doctor "+strconv.Itoa(tt.doctor), func(t *testing.T) {
			got := runSh(t, repo, HookScript, "PATH="+path, "HOME="+home, "FAKE_RC="+strconv.Itoa(tt.doctor))
			if got != tt.want {
				t.Fatalf("status = %d, want %d", got, tt.want)
			}
		})
	}
}
