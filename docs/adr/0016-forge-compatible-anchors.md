# 0016. Âncoras compatíveis com a forja

Status: aceita (fase 14). Mudança incompatível, implementada na fase
19.

## Contexto

Na versão 0.1, a âncora tirava os acentos e juntava os hífens:
"Configuração — avançada" virava `configuracao-avancada`. O GitHub
gera `configuração--avançada`. Quem escreve um vínculo com trecho
copiando da forja recebe link quebrado no guia, e o contrário também.
Com a mesma regra, o mesmo Markdown funciona nos dois lugares.

Alternativas:

- manter a regra própria;
- seguir só o GitLab, que junta os hífens repetidos;
- seguir o GitHub e aceitar também a forma do GitLab no trecho;
- aceitar no trecho também a forma antiga, sem acento.

## Decisão

A âncora segue a regra do GitHub:

- minúsculas e acentos mantidos;
- pontuação removida, inclusive `/`;
- espaço vira `-`, sem juntar hífens;
- repetição ganha `-1`, `-2`.

O trecho de um vínculo é decodificado do percent-encoding antes de
comparar. Ele casa com a âncora igual; sem âncora igual, casa na forma
do GitLab, com hífens juntados. Forma do GitLab que casa com mais de
uma âncora é erro, para o link não cair no lugar errado. A forma antiga
não é aceita: aceitá-la manteria duas regras para sempre.

O nome das páginas do guia do desenvolvedor, tirado do nome do
arquivo, continua com a regra antiga. Os nomes de arquivo já são em
kebab-case. Mudar a regra só mudaria os endereços de quem tem `_` ou
`.` no nome, sem ganho de compatibilidade.

Os testes usam títulos reais: acento, barra, travessão, crase de
código e repetição.

## Consequência

Vínculo com trecho sem acento, escrito para a regra antiga, quebra no
`check`, com `arquivo:linha`. A mudança entra no CHANGELOG em
Modificado. Os endereços de seção no guia do usuário
(`#/<página>/<âncora>`) mudam. A comparação não normaliza Unicode, porque
a biblioteca padrão não tem NFC: o título e o vínculo precisam usar a
mesma forma dos acentos, como na forja.
