package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

type Profile struct {
	Name  string
	Email string
}

func main() {

	if len(os.Args) < 3 || os.Args[1] != "profile" || os.Args[2] != "add" {
		fmt.Println("uso: gitprof profile add --name=\"nome\" --email=\"email\"")
		os.Exit(1)
	}
	profileCmd := flag.NewFlagSet("profile add", flag.ExitOnError)
	name := profileCmd.String("name", "", "nome do perfil")
	email := profileCmd.String("email", "", "email do perfil")
	profileCmd.Parse(os.Args[3:])

	if *name == "" {
		fmt.Println("Erro: A flag --name é obrigatória.")
		profileCmd.Usage()
		os.Exit(1)
	}
	if *email == "" {
		fmt.Println("Erro: A flag: --email é obrigatória")
		profileCmd.Usage()
		os.Exit(1)
	}
	if !strings.Contains(*email, "@") {
		fmt.Println("Erro: O email deve conter um @")
		profileCmd.Usage()
		os.Exit(1)
	}

	//add new profile with the values of the flags
	profile := Profile{Name: *name, Email: *email}

	fmt.Printf("Perfil %s de email: %s. Adicionado com sucesso! \n", profile.Name, profile.Email)
}
