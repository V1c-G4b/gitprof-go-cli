package main

import (
	"errors"
	"flag"
	"fmt"
	"gitprof-go-cli/internal/clone"
	"gitprof-go-cli/internal/doctor"
	"gitprof-go-cli/internal/profile"
	"gitprof-go-cli/internal/setup"
	"gitprof-go-cli/internal/store"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	if len(os.Args) >= 2 && os.Args[1] == "setup" {
		if err := cmdSetup(os.Args[2:]); err != nil {
			os.Exit(fail("setup", err))
		}
		return
	}

	if len(os.Args) < 3 || os.Args[1] != "profile" {
		usage()
		os.Exit(1)
	}
	var err error
	switch os.Args[2] {
	case "add":
		err = cmdAdd(os.Args[3:])
	case "list":
		err = cmdList(os.Args[3:])
	case "ssh":
		err = cmdConfigSSH(os.Args[3:])
	case "dir":
		err = cmdDir(os.Args[3:])
	case "key":
		err = cmdKey(os.Args[3:])
	case "user":
		err = cmdUser(os.Args[3:])
	case "clone":
		err = cmdClone(os.Args[3:])
	case "doctor":
		err = cmdDoctor(os.Args[3:])
	default:
		usage()
		os.Exit(1)
	}
	if err != nil {
		os.Exit(fail(os.Args[2], err))
	}
}

const (
	exitMismatch = 1
	exitInternal = 2
)

func fail(cmd string, err error) int {
	if cmd == "doctor" && !doctor.IsMismatch(err) {
		fmt.Fprintln(os.Stderr, "aviso: falha interna do doctor:", err)
		return exitInternal
	}
	fmt.Fprintln(os.Stderr, "erro:", err)
	return 1
}

func warn(format string, a ...any) {
	fmt.Fprintln(os.Stderr, "aviso:", fmt.Sprintf(format, a...))
}

func usage() {
	fmt.Fprintln(os.Stderr, "uso: gitprof profile <add|list|ssh|dir|key|user|clone|doctor> <args>")
	fmt.Fprintln(os.Stderr, "     gitprof setup [--dry-run | --apply]")
}

func cmdAdd(args []string) error {
	fs := flag.NewFlagSet("add", flag.ContinueOnError)
	name := fs.String("name", "", "nome do perfil")
	email := fs.String("email", "", "email do perfil")
	if err := fs.Parse(args); err != nil {
		return err
	}

	p, err := profile.New(*name, *email)
	if err != nil {
		fs.Usage()
		return err
	}

	lista, err := store.Load()
	if err != nil {
		return err
	}
	profiles, err := profile.Add(p, lista)
	if err != nil {
		return err
	}
	path, err := store.Save(profiles)
	if err != nil {
		return err
	}
	fmt.Println("Perfil salvo em", path)
	return nil
}

func cmdList(args []string) error {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	if err := fs.Parse(args); err != nil {
		return err
	}

	lista, err := store.Load()
	if err != nil {
		return err
	}

	if len(lista) == 0 {
		fmt.Println("Nenhum perfil encontrado")
		return nil
	}

	for _, p := range lista {
		fmt.Printf("%-20s %s %s %s\n", p.Name, p.Email, p.SSHHost, p.Dir)
	}
	return nil
}

func cmdConfigSSH(args []string) error {
	fs := flag.NewFlagSet("ssh", flag.ContinueOnError)
	host := fs.String("host", "", "host do ssh")
	name := fs.String("name", "", "nome do perfil")

	if err := fs.Parse(args); err != nil {
		return err
	}

	lista, err := store.Load()
	if err != nil {
		return err
	}

	profiles, err := profile.SetSSHHost(*host, lista, *name)
	if err != nil {
		return err
	}
	path, err := store.Save(profiles)
	if err != nil {
		return err
	}
	fmt.Println("SSH host salvo em", path)
	return nil
}

func cmdDir(args []string) error {
	fs := flag.NewFlagSet("dir", flag.ContinueOnError)
	name := fs.String("name", "", "nome do perfil")
	dir := fs.String("dir", "", "pasta dos repos do perfil")

	if err := fs.Parse(args); err != nil {
		return err
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	abs, err := profile.ExpandPath(*dir, home)
	if err != nil {
		return fmt.Errorf("pasta %q: %w", *dir, err)
	}

	lista, err := store.Load()
	if err != nil {
		return err
	}
	profiles, err := profile.SetDir(abs, lista, *name)
	if err != nil {
		return err
	}
	path, err := store.Save(profiles)
	if err != nil {
		return err
	}
	fmt.Println("Pasta", abs, "salva em", path)
	return nil
}

func cmdKey(args []string) error {
	fs := flag.NewFlagSet("key", flag.ContinueOnError)
	name := fs.String("name", "", "nome do perfil")
	key := fs.String("key", "", "chave privada SSH do perfil")

	if err := fs.Parse(args); err != nil {
		return err
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	abs, err := profile.ExpandPath(*key, home)
	if err != nil {
		return fmt.Errorf("chave %q: %w", *key, profile.ErrInvalidKey)
	}
	if _, err := os.Stat(abs); err != nil {
		warn("a chave %s não existe ainda; o gitprof não gera chaves", abs)
	}

	lista, err := store.Load()
	if err != nil {
		return err
	}
	profiles, err := profile.SetKey(abs, lista, *name)
	if err != nil {
		return err
	}
	path, err := store.Save(profiles)
	if err != nil {
		return err
	}
	fmt.Println("Chave", abs, "salva em", path)
	return nil
}

func cmdUser(args []string) error {
	fs := flag.NewFlagSet("user", flag.ContinueOnError)
	name := fs.String("name", "", "nome do perfil")
	user := fs.String("user", "", "user.name dos commits do perfil")

	if err := fs.Parse(args); err != nil {
		return err
	}

	lista, err := store.Load()
	if err != nil {
		return err
	}
	profiles, err := profile.SetUserName(*user, lista, *name)
	if err != nil {
		return err
	}
	path, err := store.Save(profiles)
	if err != nil {
		return err
	}
	fmt.Println("user.name salvo em", path)
	return nil
}

func cmdSetup(args []string) error {
	fs := flag.NewFlagSet("setup", flag.ContinueOnError)
	dryRun := fs.Bool("dry-run", false, "só mostra o que mudaria (padrão)")
	apply := fs.Bool("apply", false, "aplica as mudanças, com backup .bak-gitprof")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *dryRun && *apply {
		return errors.New("use --dry-run ou --apply, não os dois")
	}

	lista, err := store.Load()
	if err != nil {
		return err
	}
	if len(lista) == 0 {
		return errors.New("nenhum perfil cadastrado, rode gitprof profile add")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	env := setup.Env{Home: home}

	plan, err := setup.BuildPlan(lista, env)
	if err != nil {
		return err
	}
	for _, w := range plan.Warnings {
		warn("%s", w)
	}
	if plan.Snippet != "" {
		fmt.Fprint(os.Stderr, "\n"+plan.Snippet+"\n")
	}
	if plan.Empty() {
		fmt.Println("Nada a mudar.")
		return nil
	}
	setup.Render(os.Stdout, plan)
	if !*apply {
		fmt.Println("Nada foi alterado. Rode gitprof setup --apply para aplicar.")
		return nil
	}
	if err := setup.Apply(plan, env); err != nil {
		return err
	}
	fmt.Println("Pronto. Arquivos alterados ganharam backup com o sufixo", setup.BackupSuffix)
	return nil
}

func cmdDoctor(args []string) error {
	fs := flag.NewFlagSet("doctor", flag.ContinueOnError)
	if err := fs.Parse(args); err != nil {
		return err
	}
	email, err := doctor.CurrentEmail(".")
	if err != nil {
		return fmt.Errorf("ler user.email: %w", err)
	}
	if email == "" {
		warn("repo sem user.email, não sei conferir a identidade")
		return nil
	}
	lista, err := store.Load()
	if err != nil {
		return err
	}

	if len(lista) == 0 {
		warn("nenhum perfil cadastrado, rode gitprof profile add")
		return nil
	}

	byDir, err := matchByDir(lista)
	if err != nil {
		return err
	}
	byHost, host, hostInfo, err := matchByHost(lista)
	if err != nil {
		return err
	}

	prof, ok, err := doctor.Decide(byDir, byHost, host)
	if err != nil {
		return err
	}
	if !ok {
		warn("a pasta do repo não pertence a nenhum perfil e %s, não sei dizer qual perfil", hostInfo)
		return nil
	}

	if err := doctor.CheckEmail(prof.Name, prof.Email, email); err != nil {
		return err
	}
	fmt.Println("Email configurado corretamente")
	return nil
}

func matchByDir(lista []profile.Profile) (doctor.Match, error) {
	top, err := doctor.Toplevel(".")
	if err != nil {
		return doctor.Match{}, fmt.Errorf("achar a raiz do repo: %w", err)
	}
	top = realPath(top)
	resolved := make([]profile.Profile, len(lista))
	for i, p := range lista {
		resolved[i] = p
		if p.Dir != "" {
			resolved[i].Dir = realPath(p.Dir)
		}
	}
	p, err := profile.FindProfileByDir(top, resolved)
	if errors.Is(err, profile.ErrProfileNotFound) {
		return doctor.Match{}, nil
	}
	if err != nil {
		return doctor.Match{}, err
	}
	orig, err := profile.FindProfileByName(p.Name, lista)
	if err != nil {
		return doctor.Match{}, err
	}
	return doctor.Match{Profile: orig, Found: true}, nil
}

func matchByHost(lista []profile.Profile) (doctor.Match, string, string, error) {
	remote, err := doctor.ExpectedRemoteURL(".")
	if errors.Is(err, doctor.ErrMissingRemote) {
		return doctor.Match{}, "", "o repo não tem remote origin", nil
	}
	if err != nil {
		return doctor.Match{}, "", "", fmt.Errorf("ler remote origin: %w", err)
	}

	host, ok := doctor.HostFromRemote(remote)
	if !ok {
		return doctor.Match{}, "", "o origin não é SSH (ex.: HTTPS)", nil
	}

	p, err := profile.FindProfileBySSHHost(host, lista)
	if errors.Is(err, profile.ErrProfileNotFound) {
		return doctor.Match{}, host, fmt.Sprintf("nenhum perfil usa o alias %q", host), nil
	}
	if err != nil {
		return doctor.Match{}, "", "", err
	}
	return doctor.Match{Profile: p, Found: true}, host, "", nil
}

func realPath(p string) string {
	if r, err := filepath.EvalSymlinks(p); err == nil {
		return r
	}
	return filepath.Clean(p)
}

func cmdClone(args []string) error {
	fs := flag.NewFlagSet("clone", flag.ContinueOnError)
	rawURL := fs.String("url", "", "url do git")
	name := fs.String("name", "", "nome do perfil")

	if err := fs.Parse(args); err != nil {
		return err
	}
	lista, err := store.Load()
	if err != nil {
		return err
	}
	prof, err := profile.FindProfileByName(*name, lista)
	if err != nil {
		return err
	}
	if strings.TrimSpace(prof.SSHHost) == "" {
		return errors.New("perfil sem alias SSH, rode profile ssh")
	}

	gotURL, err := clone.RewriteURL(*rawURL, prof.SSHHost)
	if err != nil {
		return err
	}

	fmt.Println("Clonando", gotURL)

	dir, err := RepoDir(gotURL)
	if err != nil {
		return err
	}

	if err := clone.ExecClone(gotURL, dir); err != nil {
		return err
	}

	return clone.SetIdentity(dir, prof.Email)

}

func RepoDir(rawURL string) (string, error) {
	nome, err := clone.RepoName(rawURL)
	if err != nil {
		return "", err
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "git", nome), nil
}
