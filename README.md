# gitprof

CLI em Go para manter **um perfil git por conta** (pessoal, trabalho, freela...) e nunca mais commitar com a identidade errada.

O problema que ele resolve: com mais de uma conta no GitHub, é fácil clonar um repo pessoal e commitar com o email do trabalho (ou o contrário). O erro só aparece depois, no histórico publicado. O gitprof amarra cada repo a um perfil desde o clone e confere a identidade antes de cada commit.

## Como funciona

Cada perfil tem um **nome**, um **email** e um **alias SSH** (um `Host` do seu `~/.ssh/config`). O alias é o que liga o repo ao perfil:

1. `profile clone` reescreve a URL para usar o alias do perfil e já grava o `user.email` do repo.
2. `profile doctor` lê o host do `origin`, descobre o perfil dono desse alias e compara com o `user.email` do repo.
3. O hook `pre-commit` roda o doctor sozinho e bloqueia o commit se o email não bater.

## Instalação

Requer Go 1.26+ e git 2.9+.

```bash
git clone git@github.com:V1c-G4b/gitprof-go-cli.git
cd gitprof-go-cli
go build -o ~/.local/bin/gitprof .
```

Garanta que `~/.local/bin` está no seu `PATH`. (Use `go build -o`, e não `go install`: o módulo se chama `gitprof-go-cli`, então o `go install` geraria um binário com esse nome.)

## Uso

### 1. Criar os perfis

```bash
gitprof profile add -name pessoal -email teste@exemplo.com
gitprof profile add -name trabalho -email trab@exemplo.com
```

Nome e email precisam ser únicos (sem diferenciar maiúsculas).

### 2. Associar o alias SSH

```bash
gitprof profile ssh -name pessoal  -host github.com
gitprof profile ssh -name trabalho -host github-trabalho
```

O gitprof **não** edita o `~/.ssh/config`; o alias precisa existir lá, por exemplo:

```sshconfig
Host github-trabalho
  HostName github.com
  User git
  IdentityFile ~/.ssh/id_ed25519_trabalho
  IdentitiesOnly yes
```

### 3. Listar

```bash
gitprof profile listar
```

### 4. Clonar com o perfil certo

```bash
gitprof profile clone -name trabalho -url https://github.com/empresa/projeto.git
```

Aceita `git@github.com:...`, `https://github.com/...` e `http://github.com/...`. O que acontece:

- a URL vira `git@github-trabalho:empresa/projeto.git`;
- o repo é clonado em `~/git/projeto`;
- o `user.email` local do repo recebe o email do perfil.

### 5. Conferir um repo

Dentro do repo:

```bash
gitprof profile doctor
```

| Situação | Saída | Código |
|---|---|---|
| Email bate com o perfil do alias | `Email configurado corretamente` | 0 |
| Email não bate | `Email não configurado corretamente` | 1 |
| Remote HTTPS (fora do padrão `git@host:path`) | aviso | 0 |
| Repo sem `origin` | aviso | 0 |
| Nenhum perfil usa o alias do `origin` | aviso | 0 |
| Repo sem `user.email` | erro | 1 |

Os avisos saem com 0 de propósito: quando o doctor não consegue dizer qual é o perfil certo, ele não bloqueia. Só bloqueia quando tem certeza de que está errado.

## Hook pre-commit

O arquivo [`hooks/pre-commit`](hooks/pre-commit) roda o doctor antes de cada commit e, se existir, encadeia o hook local do repo (`.git/hooks/pre-commit`).

### Instalar (todos os repos)

```bash
mkdir -p ~/.config/gitprof/hooks
cp hooks/pre-commit ~/.config/gitprof/hooks/
chmod +x ~/.config/gitprof/hooks/pre-commit
git config --global core.hooksPath ~/.config/gitprof/hooks
```

Se você **já tem** um `core.hooksPath` global (`git config --global --get core.hooksPath`), não troque a pasta: adicione a chamada ao `gitprof profile doctor` dentro do `pre-commit` que já existe, senão o hook antigo para de rodar.

### Comportamento

- Sem o binário no `PATH`, o hook tenta `~/.local/bin/gitprof` (IDEs como o GoLand podem rodar sem o `PATH` do terminal). Se não achar, avisa e deixa o commit passar.
- `core.hooksPath` global substitui o `.git/hooks` de todos os repos; por isso o hook encadeia o hook local (husky e afins continuam funcionando).

### Pular ou desligar

```bash
git commit --no-verify                    # pula o hook só neste commit
git config --global --unset core.hooksPath  # desliga o hook global
```

## Onde ficam os perfis

Em `profiles.json`, dentro do diretório de configuração do sistema (`os.UserConfigDir`):

- macOS: `~/Library/Application Support/gitprof/profiles.json`
- Linux: `~/.config/gitprof/profiles.json`

## Limitações atuais

- Só GitHub é reconhecido no `clone`.
- O doctor confere apenas o `user.email` (não o `user.name`).
- O `profile ssh` não cria o alias no `~/.ssh/config`.

## Desenvolvimento

```bash
go test ./...
go vet ./...
```

Estrutura:

```
main.go             comandos da CLI
internal/profile    modelo e regras dos perfis
internal/store      leitura e gravação do profiles.json
internal/clone      reescrita de URL, clone e identidade
internal/doctor     leitura do email e do origin do repo
hooks/pre-commit    hook que chama o doctor
```
