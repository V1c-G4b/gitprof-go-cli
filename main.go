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
		fmt.Fprintln(os.Stderr, "Erro:", err)
		os.Exit(1)
	}
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
		return err
	}
	if email == "" {
		return profile.ErrEmptyEmail
	}
	lista, err := store.Load()
	if err != nil {
		return err
	}

	if len(lista) == 0 {
		fmt.Println("Nenhum perfil encontrado")
		return nil
	}

	expected, err := doctor.ExpectedRemoteURL(".")
	if errors.Is(err, doctor.ErrMissingRemote) {
		fmt.Println("aviso: repo sem remote origin, não sei dizer qual perfil")
		return nil
	}
	if err != nil {
		return err
	}

	resto, achou := strings.CutPrefix(expected, "git@")
	if !achou {
		fmt.Println("aviso: remote fora do padrão git@host:path.git")
		return nil
	}
	host, _, _ := strings.Cut(resto, ":")

	prof, err := profile.FindProfileBySSHHost(host, lista)
	if errors.Is(err, profile.ErrProfileNotFound) {
		fmt.Printf("aviso: nenhum perfil usa o alias %q, não sei dizer qual perfil\n", host)
		return nil
	}
	if err != nil {
		return err
	}

	equal := strings.EqualFold(strings.TrimSpace(prof.Email), email)
	if !equal {
		fmt.Println("Email não configurado corretamente")
		return doctor.ErrEmailMismatch
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
