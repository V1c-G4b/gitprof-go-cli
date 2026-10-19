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

O `profile ssh` só grava o alias no perfil. O alias precisa existir no `~/.ssh/config`: o `gitprof setup` (veja abaixo) gera isso para você, ou você escreve na mão, por exemplo:

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

### 4. Chave e user.name (opcional, usados pelo setup)

```bash
gitprof profile key  -name trabalho -key ~/.ssh/id_ed25519_trabalho
gitprof profile user -name trabalho -user "Seu Nome"
```

A chave é a privada, que já precisa existir: o gitprof **não gera chaves** (só avisa se o arquivo não existir). O `~` é expandido e o caminho é gravado absoluto. O `user.name` é opcional; sem ele, os commits usam o `user.name` global.

### 5. Listar

```bash
gitprof profile list
```

### 6. Clonar com o perfil certo

```bash
gitprof profile clone -name trabalho -url https://github.com/empresa/projeto.git
```

Aceita `git@github.com:...`, `https://github.com/...` e `http://github.com/...`. O que acontece:

- a URL vira `git@github-trabalho:empresa/projeto.git`;
- o repo é clonado em `~/git/projeto`;
- o `user.email` local do repo recebe o email do perfil.

### 7. Conferir um repo

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

## Setup automático

Num computador novo, depois de criar os perfis (e de copiar as chaves SSH para `~/.ssh`):

```bash
gitprof setup            # só mostra o diff do que mudaria (padrão, igual a --dry-run)
gitprof setup --apply    # aplica
```

O que o setup faz:

| O quê | Onde | Como |
|---|---|---|
| Um bloco `Host` por perfil com alias diferente de `github.com` (`HostName github.com`, `User git`, `IdentityFile` da chave, `IdentitiesOnly yes`) | `~/.ssh/config.d/gitprof` | o arquivo inteiro é do gitprof e é regerado a cada execução, não edite à mão |
| Uma única linha `Include ~/.ssh/config.d/gitprof` | topo do `~/.ssh/config` | se faltar, entra na primeira linha; se estiver duplicada ou depois de um `Host`/`Match` (onde valeria só para aquele bloco), vai para o topo. O resto do arquivo fica como está |
| `user.email`, `user.name` (se o perfil tiver) e `core.sshCommand` (se o perfil tiver chave) | `~/.config/gitprof/<perfil>.gitconfig` | `git config --file` |
| `includeIf "gitdir:<pasta>/".path` apontando para o arquivo do perfil, para cada perfil com pasta | gitconfig global | `git config --global` |
| O hook `pre-commit` e o `core.hooksPath` | `~/.config/gitprof/hooks` | só se o `core.hooksPath` estiver vazio ou já for o do gitprof |

Regras:

- **Dry-run é o padrão.** Sem `--apply`, nada é escrito. `--dry-run` e `--apply` juntos dão erro.
- **Backup:** antes de alterar um arquivo que já existe, o setup copia o original para `<arquivo>.bak-gitprof`. Se o backup já existir, ele é mantido, para guardar sempre o estado de antes do primeiro setup.
- **Escrita atômica:** os arquivos que o setup gera são escritos num temporário na mesma pasta, trocados com `rename` e seguidos de `fsync` da pasta (onde o sistema suporta); o `git config` usa o próprio lock do git.
- **Links simbólicos de arquivo:** se o `~/.ssh/config` ou o gitconfig global forem links (dotfiles, por exemplo), o setup escreve no alvo e o link continua link. O backup `.bak-gitprof` fica ao lado do alvo.
- **Idempotente:** rodar de novo sem mudar os perfis dá `Nada a mudar.`
- **Hook de terceiro:** se o `core.hooksPath` já aponta para outra pasta (husky, outro gerenciador...), o setup não mexe nele, avisa e mostra um trecho para colar no `pre-commit` dessa pasta.
- **includeIf de terceiro:** se o gitconfig global já tiver um `includeIf` para a mesma pasta de um perfil apontando para outro arquivo, o setup avisa (o que vier por último no arquivo vence) e não mexe nele.
- **Limpeza:** se um perfil perde a pasta, a chave ou o `user.name`, o setup remove a entrada correspondente que ele mesmo tinha criado.
- Perfis com alias, mas sem chave, ficam fora do `~/.ssh/config.d/gitprof`, com um aviso.
- **Links simbólicos:** o git compara o `gitdir` do `includeIf` com o caminho real do repo. Por isso o setup grava a pasta do perfil já resolvida (no macOS, `/var/...` vira `/private/var/...`; uma pasta de usuário que seja link vira o destino). Se a pasta ainda não existir, resolve a parte do caminho que já existe.
- O gitconfig global é o que o git usa: `GIT_CONFIG_GLOBAL` se estiver definido, senão `~/.gitconfig`.

## Hook pre-commit

O arquivo [`hooks/pre-commit`](hooks/pre-commit) roda o doctor antes de cada commit e, se existir, encadeia o hook local do repo (`.git/hooks/pre-commit`).

### Instalar (todos os repos)

O `gitprof setup --apply` já faz isso. Na mão:

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
- O `profile ssh` não cria o alias no `~/.ssh/config`; quem cria é o `gitprof setup`.
- O setup não gera chaves SSH nem apaga o arquivo `<perfil>.gitconfig` de um perfil removido.
- O `setup --apply` não é transacional: cada arquivo é trocado de forma atômica, mas se um passo falhar no meio, os passos anteriores já ficaram aplicados. Corrija a causa e rode de novo; como o setup é idempotente, ele só aplica o que falta.
- O `.bak-gitprof` guarda só o estado de antes do primeiro setup. As execuções seguintes não sobrescrevem esse backup, então não há histórico das versões intermediárias.

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
internal/doctor     email, origin e raiz do repo, cruzamento pasta × alias e comparação
internal/setup      gitprof setup: ssh config, gitconfig por perfil, hook, diff e backup
hooks/pre-commit    hook que chama o doctor
```
