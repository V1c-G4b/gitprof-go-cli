package setup

import (
	"fmt"
	"gitprof-go-cli/internal/profile"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type FileChange struct {
	Path    string
	Old     string
	New     string
	Exists  bool
	Mode    os.FileMode
	DirMode os.FileMode
}

type GitChange struct {
	File string
	Key  string
	Old  string
	New  string
}

type Plan struct {
	Files    []FileChange
	Git      []GitChange
	Warnings []string
	Snippet  string
}

func (p Plan) Empty() bool {
	return len(p.Files) == 0 && len(p.Git) == 0
}

func validName(name string) bool {
	return name != "" && name != "." && name != ".." &&
		!strings.ContainsAny(name, `/\`) && !strings.HasPrefix(name, ".")
}

func BuildPlan(profiles []profile.Profile, env Env) (Plan, error) {
	var plan Plan
	for _, p := range profiles {
		if !validName(p.Name) {
			return Plan{}, fmt.Errorf("nome de perfil %q não serve como nome de arquivo", p.Name)
		}
		if p.Key != "" && !fileExists(p.Key) {
			plan.Warnings = append(plan.Warnings, fmt.Sprintf("chave do perfil %q não existe em %s; o setup não gera chaves", p.Name, p.Key))
		}
	}

	fragment, warns := RenderSSHFragment(profiles)
	plan.Warnings = append(plan.Warnings, warns...)
	if err := plan.addFile(env.SSHFragment(), fragment, 0600, 0700); err != nil {
		return Plan{}, err
	}

	sshOld, _, _, err := readOptional(env.SSHConfig())
	if err != nil {
		return Plan{}, err
	}
	if err := plan.addFile(env.SSHConfig(), EnsureInclude(sshOld), 0600, 0700); err != nil {
		return Plan{}, err
	}

	if err := plan.addHooks(env); err != nil {
		return Plan{}, err
	}

	for _, p := range profiles {
		if err := plan.addProfileGitconfig(p, env); err != nil {
			return Plan{}, err
		}
	}

	if err := plan.addIncludes(profiles, env); err != nil {
		return Plan{}, err
	}
	return plan, nil
}

func (plan *Plan) addFile(path, content string, mode, dirMode os.FileMode) error {
	old, exists, curMode, err := readOptional(path)
	if err != nil {
		return err
	}
	if exists && old == content {
		return nil
	}
	if exists {
		mode = curMode
	}
	plan.Files = append(plan.Files, FileChange{Path: path, Old: old, New: content, Exists: exists, Mode: mode, DirMode: dirMode})
	return nil
}

func (plan *Plan) addGit(file, key, want string) error {
	cur, err := gitGet(file, key)
	if err != nil {
		return err
	}
	if cur == want {
		return nil
	}
	plan.Git = append(plan.Git, GitChange{File: file, Key: key, Old: cur, New: want})
	return nil
}

func (plan *Plan) addHooks(env Env) error {
	cur, err := gitGet("", "core.hooksPath")
	if err != nil {
		return err
	}
	if cur != "" {
		resolved, err := profile.ExpandPath(cur, env.Home)
		if err != nil || resolvePath(resolved) != resolvePath(env.HooksDir()) {
			plan.Warnings = append(plan.Warnings, fmt.Sprintf("core.hooksPath já aponta para %s; o setup não mexe nele. Cole o trecho abaixo no pre-commit dessa pasta", cur))
			plan.Snippet = HookSnippet
			return nil
		}
	}
	if err := plan.addFile(env.HookFile(), HookScript, 0755, 0700); err != nil {
		return err
	}
	if cur == "" {
		return plan.addGit("", "core.hooksPath", env.HooksDir())
	}
	return nil
}

func (plan *Plan) addProfileGitconfig(p profile.Profile, env Env) error {
	file := env.ProfileGitconfig(p.Name)
	sshCmd := ""
	if p.Key != "" {
		sshCmd = SSHCommand(p.Key)
	}
	for _, kv := range [][2]string{
		{"user.name", strings.TrimSpace(p.UserName)},
		{"user.email", p.Email},
		{"core.sshCommand", sshCmd},
	} {
		if err := plan.addGit(file, kv[0], kv[1]); err != nil {
			return err
		}
	}
	return nil
}

func gitdirPattern(dir string) string {
	return strings.TrimSuffix(filepath.ToSlash(resolvePath(dir)), "/") + "/"
}

func (plan *Plan) addIncludes(profiles []profile.Profile, env Env) error {
	existing, err := gitGlobalIncludes()
	if err != nil {
		return err
	}
	wanted := map[string]bool{}
	patterns := map[string]string{}
	for _, p := range profiles {
		if p.Dir == "" {
			continue
		}
		pat := gitdirPattern(p.Dir)
		key := "includeIf.gitdir:" + pat + ".path"
		wanted[strings.ToLower("includeif.gitdir:")+pat+".path"] = true
		patterns[pat] = p.Name
		if err := plan.addGit("", key, env.ProfileGitconfig(p.Name)); err != nil {
			return err
		}
	}

	for _, e := range existing {
		if !strings.HasSuffix(e.Key, ".path") {
			continue
		}
		ours := filepath.Dir(e.Value) == env.ConfigDir() && strings.HasSuffix(e.Value, ".gitconfig")
		if ours {
			if !wanted[e.Key] {
				plan.Git = append(plan.Git, GitChange{Key: e.Key, Old: e.Value})
			}
			continue
		}
		cond := strings.TrimSuffix(strings.TrimPrefix(e.Key, "includeif."), ".path")
		raw, ok := strings.CutPrefix(cond, "gitdir:")
		if !ok {
			raw, ok = strings.CutPrefix(cond, "gitdir/i:")
		}
		if !ok {
			continue
		}
		abs, err := profile.ExpandPath(raw, env.Home)
		if err != nil {
			continue
		}
		if name, clash := patterns[gitdirPattern(abs)]; clash {
			plan.Warnings = append(plan.Warnings, fmt.Sprintf("o gitconfig global já tem includeIf %q apontando para %s, na mesma pasta do perfil %q; o que vier por último no arquivo vence", cond, e.Value, name))
		}
	}
	return nil
}

func Apply(plan Plan, env Env) error {
	for _, f := range plan.Files {
		if err := os.MkdirAll(filepath.Dir(f.Path), f.DirMode); err != nil {
			return err
		}
		if f.Exists {
			if err := backup(f.Path); err != nil {
				return fmt.Errorf("backup de %s: %w", f.Path, err)
			}
		}
		if err := writeAtomic(f.Path, f.New, f.Mode); err != nil {
			return err
		}
	}

	backedUp := map[string]bool{}
	for _, c := range plan.Git {
		target := c.File
		if target == "" {
			target = env.GlobalGitconfig()
		} else if err := os.MkdirAll(filepath.Dir(target), 0700); err != nil {
			return err
		}
		if !backedUp[target] {
			if err := backup(target); err != nil {
				return fmt.Errorf("backup de %s: %w", target, err)
			}
			backedUp[target] = true
		}
		if err := gitApply(c); err != nil {
			return err
		}
	}
	return nil
}

func Render(w io.Writer, plan Plan) {
	for _, f := range plan.Files {
		if f.Exists {
			fmt.Fprintf(w, "~ %s\n", f.Path)
		} else {
			fmt.Fprintf(w, "+ %s (novo)\n", f.Path)
		}
		writeDiff(w, f.Old, f.New, 2)
		fmt.Fprintln(w)
	}
	var lastScope string
	for i, c := range plan.Git {
		scope := "git config --global"
		if c.File != "" {
			scope = "git config --file " + c.File
		}
		if i == 0 || scope != lastScope {
			fmt.Fprintln(w, scope)
			lastScope = scope
		}
		if c.Old != "" {
			fmt.Fprintf(w, "- %s = %s\n", c.Key, c.Old)
		}
		if c.New != "" {
			fmt.Fprintf(w, "+ %s = %s\n", c.Key, c.New)
		}
	}
}
