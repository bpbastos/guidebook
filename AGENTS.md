# AGENTS.md

Regras da casa para quem trabalha neste repositório, gente ou agente.

## O projeto

O guidebook é uma ferramenta de linha de comando, escrita em Go. Ela gera
guias interativos em HTML a partir de arquivos Markdown: o guia do
usuário e o guia do desenvolvedor.

É feita para quem mantém projetos: o mantenedor escreve a documentação
em Markdown no próprio repositório e a ferramenta produz os guias. Um
mesmo binário atende vários projetos, então nada aqui pode depender de
um projeto específico.

O comportamento da ferramenta está em [docs/spec.md](docs/spec.md),
escrita a partir das especificações do Bruno (mantenedor). O próprio
repositório é um projeto de entrada: o guidebook gera os guias dele
mesmo (spec, seção 13).

## Regra de origem

O guidebook foi criado para atender à necessidade de guias do usuário
em outros projetos de que o mantenedor participa. Mesmo assim, nasce do
zero: **não leia, não copie e não use como referência** código de
nenhum outro repositório desta máquina. Tudo aqui é escrito a partir
das especificações do Bruno. Na dúvida sobre se algo
conta como referência, não use e pergunte.

## Idioma

- Em inglês: identificadores, nomes de arquivo e de diretório, nomes de
  branch e chaves de configuração.
- Em pt-BR, por enquanto: todo texto para gente — documentação,
  comentários, mensagens de commit e mensagens de erro.

## Segredos

Nenhum segredo entra no repositório, nem cifrado: token, senha, chave
privada, certificado ou `.env` preenchido. Configuração local fica em
`.local/` ou `.env*`, que o `.gitignore` já exclui.

## Estrutura

- O módulo Go é `github.com/bpbastos/guidebook`.
- Diretório só nasce com conteúdo; nunca crie diretório vazio nem
  arquivo `.gitkeep` para segurar lugar.
- Saída da ferramenta fica fora do versionamento: `bin/` para o binário
  compilado e `out/` para guias gerados.

## Commits

Formato do título:

```
tipo(escopo)!: descrição
```

- `tipo`: `feat`, `fix`, `docs`, `style`, `refactor`, `test` ou `chore`.
- `(escopo)`: opcional, em minúsculas (ex.: `fix(parser): ...`).
- `!`: antes dos dois-pontos, só para mudança incompatível.
- descrição: em pt-BR, começa em minúscula, no presente do indicativo
  ("adiciona", "corrige", não "adicionar" nem "corrigiu"), sem ponto
  final.
- título com até 100 caracteres.

Havendo corpo: uma linha em branco depois do título e corpo quebrado em
72 colunas.

Sem atribuição de IA em mensagem nenhuma: nada de nome de ferramenta de
IA, `Co-Authored-By` de assistente, "generated with" ou emoji de robô.

Quem garante é o hook `.githooks/commit-msg`, ativado com `make hooks`.
Ele lista todas as violações de uma vez. Mensagens de merge, revert,
`fixup!`, `squash!` e `amend!` ficam fora da checagem de formato, mas
não da de atribuição. O tempo verbal é checado por heurística: o hook
recusa a primeira palavra com cara de infinitivo, passado, gerúndio ou
particípio. Se a heurística errar com uma palavra legítima, avise o
Bruno para ajustar o hook. Nunca contorne com `--no-verify`.

## Ciclo de trabalho

O trabalho segue as fases de `docs/spec.md` (seção 12): uma fase, uma
branch, um pull request.

1. Branch a partir da `main` atualizada (passo 6), com prefixo
   `chore/`, `docs/`, `feat/` ou `fix/` e nome em inglês e kebab-case
   (ex.: `feat/markdown-rules`).
2. Commits na branch, seguindo a convenção acima. Em fase com código,
   TDD: o teste falha pelo motivo certo antes da implementação. Fase
   que muda comportamento atualiza também a página do guia do próprio
   guidebook (`docs/user/`), e o `guidebook check` da CI confere.
3. Push só da branch, por SSH (`origin` é
   `git@github.com:bpbastos/guidebook.git`). Nunca push da `main`,
   nunca com força (`--force`, `--force-with-lease`, `+refspec`).
4. Logo depois do push, o agente avisa o conselheiro por `SendMessage`.
   O nome atual da sessão do conselheiro vem do Bruno ou do
   `ListAgents`, porque muda quando a sessão reinicia. O aviso traz
   branch, SHA do último commit, resumo, testes rodados e decisões
   tomadas. Nos testes, diz qual ficou vermelho antes do verde e por
   quê. Depois o agente encerra o turno e espera. Sem `SendMessage`
   disponível, avisa isso no resumo ao Bruno e espera que ele cole a
   resposta.
5. O conselheiro revisa, abre o PR e mescla por squash; a `main` é
   protegida no GitHub e só recebe assim. A resposta é "mesclado" (segue
   para a próxima fase) ou uma lista de ajustes (corrige na mesma
   branch, faz push e avisa de novo). O agente não tem token no GitHub e
   não abre pull request.
6. A `main` local só anda com:

   ```sh
   git fetch origin
   git switch main
   git merge --ff-only origin/main
   ```

   Nada de commit, merge comum ou rebase direto na `main`.

## Autonomia

O agente decide sozinho, dentro destas regras e da spec: estrutura de
pacotes e arquivos, nomes, redação e detalhes que a spec não fixa. Toda
decisão com alternativa real vira um ADR curto em
`docs/adr/NNNN-titulo.md` (contexto, decisão, consequência).

Já aprovadas:

- goldmark e `go.yaml.in/yaml/v3`, no Go, com versões fixas;
- MiniSearch e Mermaid vendorizados, no navegador, sem CDN;
- GitHub Actions, com actions fixadas por SHA;
- Docker, com imagens base fixadas por digest;
- publicação no GitHub Container Registry pela CI, só em tag de versão
  criada pelo Bruno.

Toda dúvida ou parada vai primeiro ao conselheiro, por `SendMessage`,
não por pergunta no terminal. O conselheiro responde o que é técnico e
leva ao Bruno o que é dele, voltando com a decisão. Pergunta no terminal
só se o `SendMessage` falhar.

São do Bruno, e o agente para até ter a decisão:

- dependência fora da lista acima, no `go.mod` ou no navegador
  (JavaScript, CSS, fonte);
- ferramenta nova no fluxo fora da lista acima;
- qualquer mudança nesta convenção, incluindo o hook;
- qualquer coisa externa ao repositório (outros diretórios da máquina,
  serviços, contas, configuração global);
- contradição na especificação;
- código próprio acima do teto (Go sem testes ~4.000 linhas,
  JavaScript ~1.500).
