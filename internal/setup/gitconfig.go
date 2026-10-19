package setup

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

func scopeArgs(file string) []string {
	if file == "" {
		return []string{"config", "--global"}
	}
	return []string{"config", "--file", file}
}

func gitGet(file, key string) (string, error) {
	args := append(scopeArgs(file), "--get", key)
	out, err := exec.Command("git", args...).Output()
	if err != nil {
		var exitError *exec.ExitError
		if errors.As(err, &exitError) && exitError.ExitCode() == 1 {
			return "", nil
		}
		return "", fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return strings.TrimSuffix(string(out), "\n"), nil
}

type gitEntry struct {
	Key   string
	Value string
}

func gitGlobalIncludes() ([]gitEntry, error) {
	args := append(scopeArgs(""), "--null", "--get-regexp", `^includeif\.`)
	out, err := exec.Command("git", args...).Output()
	if err != nil {
		var exitError *exec.ExitError
		if errors.As(err, &exitError) && exitError.ExitCode() == 1 {
			return nil, nil
		}
		return nil, fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	var entries []gitEntry
	for _, rec := range strings.Split(string(out), "\x00") {
		if rec == "" {
			continue
		}
		key, value, _ := strings.Cut(rec, "\n")
		entries = append(entries, gitEntry{Key: key, Value: value})
	}
	return entries, nil
}

func gitApply(c GitChange) error {
	args := scopeArgs(c.File)
	if c.New == "" {
		args = append(args, "--unset-all", c.Key)
	} else {
		args = append(args, "--replace-all", c.Key, c.New)
	}
	out, err := exec.Command("git", args...).CombinedOutput()
	if err != nil {
		var exitError *exec.ExitError
		if c.New == "" && errors.As(err, &exitError) && exitError.ExitCode() == 5 {
			return nil
		}
		return fmt.Errorf("git %s: %v: %s", strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return nil
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

func SSHCommand(key string) string {
	return "ssh -i " + shellQuote(key) + " -o IdentitiesOnly=yes"
}
