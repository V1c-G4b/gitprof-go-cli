package setup

import (
	"bytes"
	"gitprof-go-cli/internal/profile"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func tempHome(t *testing.T) Env {
	t.Helper()
	base := t.TempDir()
	real := filepath.Join(base, "real")
	if err := os.Mkdir(real, 0700); err != nil {
		t.Fatal(err)
	}
	home := filepath.Join(base, "home")
	if err := os.Symlink(real, home); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("GIT_CONFIG_GLOBAL", filepath.Join(home, "global.gitconfig"))
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	return Env{Home: home}
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
}

func mustRead(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func git(t *testing.T, args ...string) string {
	t.Helper()
	out, err := exec.Command("git", args...).CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v: %s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

func perfis(env Env) []profile.Profile {
	return []profile.Profile{
		{Name: "pessoal", Email: "teste@exemplo.com", SSHHost: "github.com", Dir: filepath.Join(env.Home, "dev", "pessoal")},
		{Name: "trabalho", Email: "trab@exemplo.com", SSHHost: "github-trab", Dir: filepath.Join(env.Home, "dev", "work"), Key: filepath.Join(env.Home, ".ssh", "id_trab"), UserName: "Teste Trab"},
	}
}

func applyOnce(t *testing.T, pl []profile.Profile, env Env) Plan {
	t.Helper()
	plan, err := BuildPlan(pl, env)
	if err != nil {
		t.Fatal(err)
	}
	if err := Apply(plan, env); err != nil {
		t.Fatal(err)
	}
	return plan
}

func TestEnsureInclude(t *testing.T) {
	tests := []struct {
		nome string
		in   string
		want string
	}{
		{"arquivo vazio", "", IncludeLine + "\n"},
		{"sem include", "Host a\n  User git\n", IncludeLine + "\n\nHost a\n  User git\n"},
		{"sem newline no fim", "Host a", IncludeLine + "\n\nHost a\n"},
		{"ja no topo", IncludeLine + "\n\nHost a\n", IncludeLine + "\n\nHost a\n"},
		{"depois de opcao global mas antes de host", "AddKeysToAgent yes\n" + IncludeLine + "\nHost a\n", "AddKeysToAgent yes\n" + IncludeLine + "\nHost a\n"},
		{"dentro de um host vai para o topo", "Host a\n  User git\n" + IncludeLine + "\n", IncludeLine + "\nHost a\n  User git\n"},
		{"depois de match vai para o topo", "Match host x\n  User git\n" + IncludeLine + "\n", IncludeLine + "\nMatch host x\n  User git\n"},
		{"duplicado vira um", IncludeLine + "\n" + IncludeLine + "\nHost a\n", IncludeLine + "\nHost a\n"},
		{"variante relativa conta", "Include config.d/gitprof\nHost a\n", "Include config.d/gitprof\nHost a\n"},
		{"include de outro arquivo fica", "Include ~/.ssh/outro\nHost a\n", IncludeLine + "\n\nInclude ~/.ssh/outro\nHost a\n"},
	}
	for _, tt := range tests {
		t.Run(tt.nome, func(t *testing.T) {
			got := EnsureInclude(tt.in)
			if got != tt.want {
				t.Fatalf("EnsureInclude(%q)\n got %q\nwant %q", tt.in, got, tt.want)
			}
			if again := EnsureInclude(got); again != got {
				t.Fatalf("segunda passada mudou: %q", again)
			}
		})
	}
}

func TestRenderSSHFragment(t *testing.T) {
	pl := []profile.Profile{
		{Name: "pessoal", Email: "teste@exemplo.com", SSHHost: "github.com", Key: "/k/pessoal"},
		{Name: "trabalho", Email: "trab@exemplo.com", SSHHost: "github-trab", Key: "/k/trab"},
		{Name: "semchave", Email: "semchave@exemplo.com", SSHHost: "github-sem"},
		{Name: "repetido", Email: "rep@exemplo.com", SSHHost: "GitHub-Trab", Key: "/k/rep"},
		{Name: "espaco", Email: "esp@exemplo.com", SSHHost: "github-esp", Key: "/k/com espaco"},
		{Name: "semalias", Email: "semalias@exemplo.com"},
	}
	got, warns := RenderSSHFragment(pl)
	want := "Host github-trab\n  HostName github.com\n  User git\n  IdentityFile /k/trab\n  IdentitiesOnly yes\n" +
		"\nHost github-esp\n  HostName github.com\n  User git\n  IdentityFile \"/k/com espaco\"\n  IdentitiesOnly yes\n"
	if got != want {
		t.Fatalf("got\n%s\nwant\n%s", got, want)
	}
	if len(warns) != 2 || !strings.Contains(warns[0], `"semchave"`) || !strings.Contains(warns[1], `"repetido"`) {
		t.Fatalf("avisos inesperados: %q", warns)
	}
}

func TestDryRunNaoMexe(t *testing.T) {
	env := tempHome(t)
	plan, err := BuildPlan(perfis(env), env)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Empty() {
		t.Fatal("plano vazio num HOME novo")
	}
	var out bytes.Buffer
	Render(&out, plan)
	for _, parte := range []string{env.SSHFragment(), env.SSHConfig(), "+ " + IncludeLine, "+ user.email = trab@exemplo.com", "includeIf.gitdir:"} {
		if !strings.Contains(out.String(), parte) {
			t.Fatalf("diff sem %q:\n%s", parte, out.String())
		}
	}
	entries, err := os.ReadDir(env.Home)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("dry-run criou arquivos: %v", entries)
	}
}

func TestApplyIdempotente(t *testing.T) {
	env := tempHome(t)
	original := "Host antigo\n  HostName exemplo.com\n"
	mustWrite(t, env.SSHConfig(), original)
	mustWrite(t, filepath.Join(env.Home, ".ssh", "id_trab"), "chave falsa")
	mustWrite(t, env.GlobalGitconfig(), "[user]\n\tname = Teste Global\n")
	pl := perfis(env)

	applyOnce(t, pl, env)

	second, err := BuildPlan(pl, env)
	if err != nil {
		t.Fatal(err)
	}
	if !second.Empty() {
		var out bytes.Buffer
		Render(&out, second)
		t.Fatalf("segunda execução ainda tem mudanças:\n%s", out.String())
	}

	sshCfg := mustRead(t, env.SSHConfig())
	if !strings.HasPrefix(sshCfg, IncludeLine+"\n") || strings.Count(sshCfg, IncludeLine) != 1 || !strings.Contains(sshCfg, original) {
		t.Fatalf("~/.ssh/config inesperado:\n%s", sshCfg)
	}
	if got := mustRead(t, env.SSHConfig()+BackupSuffix); got != original {
		t.Fatalf("backup do ssh config = %q, want %q", got, original)
	}
	if got := mustRead(t, env.GlobalGitconfig()+BackupSuffix); !strings.Contains(got, "Teste Global") {
		t.Fatalf("backup do gitconfig global = %q", got)
	}
	frag := mustRead(t, env.SSHFragment())
	if !strings.Contains(frag, "Host github-trab") || strings.Contains(frag, "Host github.com") {
		t.Fatalf("fragmento inesperado:\n%s", frag)
	}
	if mustRead(t, env.HookFile()) != HookScript {
		t.Fatal("hook instalado diferente do HookScript")
	}
	if info, err := os.Stat(env.HookFile()); err != nil || info.Mode().Perm()&0100 == 0 {
		t.Fatalf("hook sem permissão de execução: %v", err)
	}
	if got := git(t, "config", "--global", "core.hooksPath"); got != env.HooksDir() {
		t.Fatalf("core.hooksPath = %q", got)
	}
	trab := env.ProfileGitconfig("trabalho")
	if got := git(t, "config", "--file", trab, "user.email"); got != "trab@exemplo.com" {
		t.Fatalf("user.email do perfil = %q", got)
	}
	if got := git(t, "config", "--file", trab, "user.name"); got != "Teste Trab" {
		t.Fatalf("user.name do perfil = %q", got)
	}
	if got := git(t, "config", "--file", trab, "core.sshCommand"); got != SSHCommand(filepath.Join(env.Home, ".ssh", "id_trab")) {
		t.Fatalf("core.sshCommand = %q", got)
	}

	repo := filepath.Join(env.Home, "dev", "work", "repo")
	git(t, "init", "-q", repo)
	if got := git(t, "-C", repo, "config", "user.email"); got != "trab@exemplo.com" {
		t.Fatalf("repo em dev/work usa %q, want trab@exemplo.com", got)
	}
	outro := filepath.Join(env.Home, "dev", "work2", "repo")
	git(t, "init", "-q", outro)
	if got, _ := exec.Command("git", "-C", outro, "config", "user.email").Output(); strings.TrimSpace(string(got)) == "trab@exemplo.com" {
		t.Fatal("repo em dev/work2 herdou o email do trabalho")
	}
}

func TestApplyPreservaBackupOriginal(t *testing.T) {
	env := tempHome(t)
	mustWrite(t, env.SSHConfig(), "Host original\n")
	pl := perfis(env)[:1]
	applyOnce(t, pl, env)
	mustWrite(t, env.SSHConfig(), "Host editado\n")
	applyOnce(t, pl, env)
	if got := mustRead(t, env.SSHConfig()+BackupSuffix); got != "Host original\n" {
		t.Fatalf("backup = %q, want o original", got)
	}
}

func TestHooksPathDeTerceiro(t *testing.T) {
	env := tempHome(t)
	git(t, "config", "--global", "core.hooksPath", "/outro/hooks")
	plan := applyOnce(t, perfis(env)[:1], env)

	if plan.Snippet == "" {
		t.Fatal("sem trecho para colar")
	}
	if !strings.Contains(strings.Join(plan.Warnings, "\n"), "/outro/hooks") {
		t.Fatalf("avisos sem o hooksPath atual: %q", plan.Warnings)
	}
	if fileExists(env.HookFile()) {
		t.Fatal("instalou o hook mesmo com hooksPath de terceiro")
	}
	if got := git(t, "config", "--global", "core.hooksPath"); got != "/outro/hooks" {
		t.Fatalf("core.hooksPath mudou para %q", got)
	}
}

func TestHooksPathDoGitprofComTil(t *testing.T) {
	env := tempHome(t)
	git(t, "config", "--global", "core.hooksPath", "~/.config/gitprof/hooks")
	plan, err := BuildPlan(perfis(env)[:1], env)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Snippet != "" {
		t.Fatal("tratou o hooksPath do próprio gitprof como de terceiro")
	}
	for _, c := range plan.Git {
		if c.Key == "core.hooksPath" {
			t.Fatalf("quis trocar o hooksPath: %+v", c)
		}
	}
}

func TestRemoveIncludeIfOrfao(t *testing.T) {
	env := tempHome(t)
	pl := perfis(env)
	applyOnce(t, pl, env)

	key := "includeIf.gitdir:" + gitdirPattern(filepath.Join(env.Home, "dev", "work")) + ".path"
	if got := git(t, "config", "--global", "--get", key); got != env.ProfileGitconfig("trabalho") {
		t.Fatalf("includeIf antes da remoção = %q", got)
	}

	pl[1].Dir = ""
	pl[1].Key = ""
	pl[1].UserName = ""
	applyOnce(t, pl, env)

	if out, _ := exec.Command("git", "config", "--global", "--get", key).Output(); len(out) != 0 {
		t.Fatalf("includeIf órfão continua: %s", out)
	}
	trab := env.ProfileGitconfig("trabalho")
	for _, k := range []string{"core.sshCommand", "user.name"} {
		if out, _ := exec.Command("git", "config", "--file", trab, "--get", k).Output(); len(out) != 0 {
			t.Fatalf("%s continua no perfil: %s", k, out)
		}
	}
	again, err := BuildPlan(pl, env)
	if err != nil {
		t.Fatal(err)
	}
	if !again.Empty() {
		t.Fatalf("não ficou idempotente: %+v", again)
	}
}

func TestIncludeIfDeTerceiroAvisa(t *testing.T) {
	env := tempHome(t)
	git(t, "config", "--global", "includeIf.gitdir:~/dev/work/.path", "/outro/trab.gitconfig")
	plan, err := BuildPlan(perfis(env), env)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(plan.Warnings, "\n"), "/outro/trab.gitconfig") {
		t.Fatalf("sem aviso do includeIf de terceiro: %q", plan.Warnings)
	}
	for _, c := range plan.Git {
		if c.Old == "/outro/trab.gitconfig" {
			t.Fatalf("quis mexer no includeIf de terceiro: %+v", c)
		}
	}
}

func TestNomeDePerfilInvalido(t *testing.T) {
	env := tempHome(t)
	for _, nome := range []string{"../x", "a/b", ".oculto", ".."} {
		_, err := BuildPlan([]profile.Profile{{Name: nome, Email: "teste@exemplo.com"}}, env)
		if err == nil {
			t.Fatalf("aceitou o nome %q", nome)
		}
	}
}

func TestHookScriptIgualAoArquivo(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "hooks", "pre-commit"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != HookScript {
		t.Fatal("HookScript diverge de hooks/pre-commit")
	}
}

func TestSSHCommandQuoting(t *testing.T) {
	got := SSHCommand("/k/it's")
	want := `ssh -i '/k/it'\''s' -o IdentitiesOnly=yes`
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestResolvePath(t *testing.T) {
	env := tempHome(t)
	real, err := filepath.EvalSymlinks(env.Home)
	if err != nil {
		t.Fatal(err)
	}
	if real == env.Home {
		t.Fatal("HOME do teste deveria ser um link simbólico")
	}
	mustWrite(t, filepath.Join(env.Home, "dev", "work", "x"), "")
	tests := []struct {
		nome string
		in   string
		want string
	}{
		{"home", env.Home, real},
		{"pasta que existe", filepath.Join(env.Home, "dev", "work"), filepath.Join(real, "dev", "work")},
		{"pasta que ainda não existe", filepath.Join(env.Home, "dev", "novo", "sub"), filepath.Join(real, "dev", "novo", "sub")},
		{"sem link", filepath.Join(real, "dev"), filepath.Join(real, "dev")},
	}
	for _, tt := range tests {
		t.Run(tt.nome, func(t *testing.T) {
			if got := resolvePath(tt.in); got != tt.want {
				t.Fatalf("resolvePath(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestIncludeIfUsaCaminhoReal(t *testing.T) {
	env := tempHome(t)
	real, err := filepath.EvalSymlinks(env.Home)
	if err != nil {
		t.Fatal(err)
	}
	applyOnce(t, perfis(env), env)
	out := git(t, "config", "--global", "--name-only", "--get-regexp", `^includeif\.`)
	want := "includeif.gitdir:" + filepath.ToSlash(filepath.Join(real, "dev", "work")) + "/.path"
	if !strings.Contains(out, want) {
		t.Fatalf("includeIf sem o caminho real:\n%s\nwant %s", out, want)
	}
	if strings.Contains(out, filepath.ToSlash(env.Home)+"/") {
		t.Fatalf("includeIf ainda usa o caminho do link:\n%s", out)
	}
}

func TestHooksPathPeloCaminhoReal(t *testing.T) {
	env := tempHome(t)
	real, err := filepath.EvalSymlinks(env.Home)
	if err != nil {
		t.Fatal(err)
	}
	git(t, "config", "--global", "core.hooksPath", filepath.Join(real, ".config", "gitprof", "hooks"))
	plan, err := BuildPlan(perfis(env)[:1], env)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Snippet != "" {
		t.Fatal("tratou o hooksPath do gitprof pelo caminho real como de terceiro")
	}
}

func assertLink(t *testing.T, link, target string) {
	t.Helper()
	info, err := os.Lstat(link)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("%s deixou de ser link", link)
	}
	dest, err := os.Readlink(link)
	if err != nil {
		t.Fatal(err)
	}
	if dest != target {
		t.Fatalf("%s aponta para %s, want %s", link, dest, target)
	}
}

func TestApplyPreservaSymlinks(t *testing.T) {
	env := tempHome(t)
	dotfiles := filepath.Join(env.Home, "dotfiles")
	sshTarget := filepath.Join(dotfiles, "ssh_config")
	gitTarget := filepath.Join(dotfiles, "gitconfig")
	mustWrite(t, sshTarget, "Host antigo\n  HostName exemplo.com\n")
	mustWrite(t, gitTarget, "[user]\n\tname = Teste Global\n")
	if err := os.MkdirAll(filepath.Join(env.Home, ".ssh"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(sshTarget, env.SSHConfig()); err != nil {
		t.Fatal(err)
	}
	gitLink := filepath.Join(env.Home, ".gitconfig")
	if err := os.Symlink(gitTarget, gitLink); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GIT_CONFIG_GLOBAL", gitLink)

	pl := perfis(env)
	applyOnce(t, pl, env)

	assertLink(t, env.SSHConfig(), sshTarget)
	assertLink(t, gitLink, gitTarget)

	sshCfg := mustRead(t, sshTarget)
	if !strings.HasPrefix(sshCfg, IncludeLine+"\n") || !strings.Contains(sshCfg, "Host antigo") {
		t.Fatalf("alvo do ssh config sem o Include:\n%s", sshCfg)
	}
	if got := mustRead(t, sshTarget+BackupSuffix); got != "Host antigo\n  HostName exemplo.com\n" {
		t.Fatalf("backup do alvo do ssh config = %q", got)
	}
	if fileExists(env.SSHConfig() + BackupSuffix) {
		t.Fatal("backup foi parar ao lado do link, não do alvo")
	}
	gitCfg := mustRead(t, gitTarget)
	if !strings.Contains(gitCfg, "hooksPath") || !strings.Contains(gitCfg, "includeIf") || !strings.Contains(gitCfg, "Teste Global") {
		t.Fatalf("alvo do gitconfig global sem as mudanças:\n%s", gitCfg)
	}
	if got := mustRead(t, gitTarget+BackupSuffix); !strings.Contains(got, "Teste Global") || strings.Contains(got, "includeIf") {
		t.Fatalf("backup do alvo do gitconfig = %q", got)
	}

	again, err := BuildPlan(pl, env)
	if err != nil {
		t.Fatal(err)
	}
	if !again.Empty() {
		t.Fatalf("não ficou idempotente com links: %+v", again)
	}
}

func TestWriteAtomicLinkQuebrado(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "alvo")
	link := filepath.Join(dir, "link")
	if err := os.Symlink("alvo", link); err != nil {
		t.Fatal(err)
	}
	if err := writeAtomic(link, "conteudo\n", 0600); err != nil {
		t.Fatal(err)
	}
	assertLink(t, link, "alvo")
	if got := mustRead(t, target); got != "conteudo\n" {
		t.Fatalf("alvo = %q", got)
	}
}

func TestWriteAtomicSemSobras(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "arquivo")
	for range 2 {
		if err := writeAtomic(path, "x\n", 0640); err != nil {
			t.Fatal(err)
		}
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("sobrou temporário: %v", entries)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0640 {
		t.Fatalf("modo = %v, want 0640", info.Mode().Perm())
	}
}
