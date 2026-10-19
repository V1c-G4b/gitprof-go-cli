# gitprof

CLI em Go para manter **um perfil git por conta** (pessoal, trabalho, freela...) e nunca mais commitar com a identidade errada.

O problema que ele resolve: com mais de uma conta no GitHub, é fácil clonar um repo pessoal e commitar com o email do trabalho (ou o contrário). O erro só aparece depois, no histórico publicado. O gitprof amarra cada repo a um perfil desde o clone e confere a identidade antes de cada commit.

## Como funciona

Cada perfil tem um **nome**, um **email** e, opcionalmente, um **alias SSH** (um `Host` do seu `~/.ssh/config`) e uma **pasta** (onde ficam os repos daquele perfil). O alias e a pasta são o que ligam o repo ao perfil:

1. `profile clone` reescreve a URL para usar o alias do perfil e já grava o `user.email` do repo.
2. `profile doctor` descobre o perfil pela pasta do repo e pelo host do `origin`, confere se os dois concordam e compara com o `user.email` do repo.
3. O hook `pre-commit` roda o doctor sozinho e bloqueia o commit se o perfil ou o email não baterem.

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

### 3. Associar uma pasta (opcional)

```bash
gitprof profile dir -name trabalho -dir ~/dev/work
```

Todo repo dentro de `~/dev/work` passa a ser do perfil `trabalho`. O `~` é expandido e o caminho é gravado absoluto e limpo (sem `/` no fim, sem `..`). Um caminho relativo é resolvido a partir da pasta atual.

### 4. Listar

```bash
gitprof profile list
```

### 5. Clonar com o perfil certo

```bash
gitprof profile clone -name trabalho -url https://github.com/empresa/projeto.git
```

Aceita `git@github.com:...`, `https://github.com/...` e `http://github.com/...`. O que acontece:

- a URL vira `git@github-trabalho:empresa/projeto.git`;
- o repo é clonado em `~/git/projeto`;
- o `user.email` local do repo recebe o email do perfil.

### 6. Conferir um repo

Dentro do repo:

```bash
gitprof profile doctor
```

O doctor procura o perfil de dois jeitos:

- **pela pasta**: a raiz do repo (`git rev-parse --show-toplevel`) precisa estar dentro do `dir` de um perfil. Se mais de um perfil casar, ganha a pasta mais específica (o prefixo mais longo). A comparação respeita o separador de caminho, então `~/dev/work2` não casa com `~/dev/work`. Links simbólicos são resolvidos dos dois lados;
- **pelo alias**: o host do `origin` precisa ser o `ssh_host` de um perfil.

| Pasta | Alias | Perfil usado |
|---|---|---|
| perfil A | perfil A | A |
| perfil A | perfil B | nenhum: **conflito, código 1** |
| perfil A | nada | A |
| nada | perfil B | B |
| nada | nada | nenhum: aviso, código 0 |

Com o perfil escolhido, o doctor compara o email dele com o `user.email` do repo.

| Situação | Saída | Código |
|---|---|---|
| Email bate com o perfil | `Email configurado corretamente` | 0 |
| Nem a pasta nem o alias indicam um perfil (remote HTTPS, sem `origin`, alias desconhecido...) | aviso | 0 |
| Nenhum perfil cadastrado | aviso | 0 |
| Repo sem `user.email` | aviso | 0 |
| A pasta e o alias apontam para perfis diferentes | erro com os dois perfis, a pasta e o alias | **1** |
| Email não bate | erro com o perfil, o email esperado e o atual | **1** |
| Falha interna (`profiles.json` ilegível, erro do git...) | aviso de falha interna | 2 |

O host vem do `origin` nos formatos `git@host:path`, `user@host:path` e `ssh://[user@]host[:porta]/path`; o host pode ser um alias do `~/.ssh/config`.

Os códigos são um contrato com o hook: **só o 1 bloqueia o commit**. Quando o doctor não consegue dizer qual é o perfil certo (avisos, 0) ou quando ele mesmo falha (2), não bloqueia. Só bloqueia quando tem certeza de que está errado. Avisos e erros saem no stderr.

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

- Doctor com código 1 (email errado ou pasta e alias de perfis diferentes): o commit é bloqueado.
- Doctor com código 2 (falha interna) ou outro código: o hook avisa e deixa o commit passar.
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
internal/doctor     email e origin do repo, host do remote e comparação
hooks/pre-commit    hook que chama o doctor
```
