# 0012. O serve só serve o guia do usuário

Status: aceita (fase 12).

## Contexto

O Bruno decidiu que o guia do desenvolvedor nunca é servido, como regra
fixa e sem opção. O guia do desenvolvedor existe só como arquivo local,
gerado com `guidebook build`. Faltava escolher como o `serve` cumpre a
regra: só não gerar o guia, ou também recusar os arquivos dele.

## Decisão

Três camadas, cada uma suficiente sozinha:

- **Leitura.** O `serve` lê o `guidebook.yaml` com `config.LoadUser`,
  que descarta a seção `developer` antes de validar. As fontes dela não
  são lidas, conferidas nem varridas.
- **Build.** O build do `serve` grava só o guia do usuário
  (`build.Options.UserOnly`).
- **Resposta.** Um caminho cujo último nome, sem diferença de caixa, é
  `developer-guide.html` ou `developer-data.js` dá 404 em qualquer
  pasta. A conferência vem depois do `path.Clean`, então `..`, barras
  repetidas e caminho codificado chegam ao mesmo nome.

A alternativa de só não gerar o guia foi descartada: um arquivo que
aparecesse no diretório por engano seria servido.

## Consequência

Projeto só com `developer` não tem o que servir: o `serve` recusa ao
subir, com código 2. O compose monta só o que o guia do usuário lê, e a
CI confere que `/developer-guide.html` dá 404. A recusa por nome
alcança também um arquivo do guia do usuário chamado
`developer-guide.html`, o que não tem uso legítimo.
