package main

import (
	"errors"
	"flag"
	"fmt"
	"gitprof-go-cli/internal/clone"
	"gitprof-go-cli/internal/profile"
	"gitprof-go-cli/internal/store"
	"os"
	"path/filepath"
	"strings"
)

func main() {

	if len(os.Args) < 3 || os.Args[1] != "profile" {
		fmt.Println("Uso: gitprof profile <add|listar|ssh|clone> <args>")
		os.Exit(1)
	}
	var err error
	switch os.Args[2] {
	case "add":
		err = cmdAdd(os.Args[3:])
	case "listar":
		err = cmdListar(os.Args[3:])
	case "ssh":
		err = cmdConfigSSH(os.Args[3:])
	case "clone":
		err = cmdClone(os.Args[3:])
	default:
		fmt.Fprintln(os.Stderr, "Uso: gitprof profile <add|listar|ssh|clone> <args>")
		os.Exit(1)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "Erro:", err)
		os.Exit(1)
	}
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
		fmt.Fprintln(os.Stderr, err)
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

func cmdListar(args []string) error {
	fs := flag.NewFlagSet("listar", flag.ContinueOnError)
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
