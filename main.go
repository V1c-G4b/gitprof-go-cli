package main

import (
	"errors"
	"flag"
	"fmt"
	"gitprof-go-cli/internal/clone"
	"gitprof-go-cli/internal/doctor"
	"gitprof-go-cli/internal/profile"
	"gitprof-go-cli/internal/store"
	"os"
	"path/filepath"
	"strings"
)

func main() {

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

// Códigos de saída do doctor. Viraram contrato com o hook pre-commit, que
// roda em todos os repos: só exitMismatch bloqueia o commit.
const (
	exitMismatch = 1 // email do repo não bate com o perfil do alias
	exitInternal = 2 // falha da própria ferramenta; o hook avisa e deixa passar
)

// fail imprime o erro (uma vez só, aqui no main) e decide o código de saída.
// No doctor, tudo que não é mismatch confirmado é falha interna. Os outros
// comandos continuam saindo com 1.
func fail(cmd string, err error) int {
	if cmd == "doctor" && !errors.Is(err, doctor.ErrEmailMismatch) {
		fmt.Fprintln(os.Stderr, "aviso: falha interna do doctor:", err)
		return exitInternal
	}
	fmt.Fprintln(os.Stderr, "erro:", err)
	return 1 // no doctor, 1 é sempre exitMismatch
}

// warn mostra um aviso no stderr. Aviso não bloqueia: o doctor sai com 0.
func warn(format string, a ...any) {
	fmt.Fprintln(os.Stderr, "aviso:", fmt.Sprintf(format, a...))
}

func usage() {
	fmt.Fprintln(os.Stderr, "uso: gitprof profile <add|list|ssh|clone|doctor> <args>")
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
		fmt.Printf("%-20s %s %s \n", p.Name, p.Email, p.SSHHost)
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

	remote, err := doctor.ExpectedRemoteURL(".")
	if errors.Is(err, doctor.ErrMissingRemote) {
		warn("repo sem remote origin, não sei dizer qual perfil")
		return nil
	}
	if err != nil {
		return fmt.Errorf("ler remote origin: %w", err)
	}

	host, ok := doctor.HostFromRemote(remote)
	if !ok {
		warn("remote origin não é SSH (ex.: HTTPS), não sei dizer qual perfil")
		return nil
	}

	prof, err := profile.FindProfileBySSHHost(host, lista)
	if errors.Is(err, profile.ErrProfileNotFound) {
		warn("nenhum perfil usa o alias %q, não sei dizer qual perfil", host)
		return nil
	}
	if err != nil {
		return err
	}

	if err := doctor.CheckEmail(prof.Name, prof.Email, email); err != nil {
		return err
	}
	fmt.Println("Email configurado corretamente")
	return nil
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
