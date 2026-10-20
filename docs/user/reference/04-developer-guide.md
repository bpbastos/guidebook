# Guia do desenvolvedor

O guia do desenvolvedor reúne a documentação técnica do projeto. Cada
seção vem de arquivos que você escolhe no `guidebook.yaml`.

Ele existe só como arquivo local. Quem tem acesso ao repositório gera o
guia com `guidebook build`; o `guidebook serve` nunca o serve.

## Seções

```yaml
developer:
  repository_url: https://github.com/example/pedido-facil/blob/main
  sections:
    overview:
      sources: [docs/developer/overview.md]
    modules:
      sources: [docs/developer/modules/*.md]
    business_rules:
      sources: [docs/developer/rules/*.md]
    decisions:
      title: Decisões de arquitetura
      sources: [docs/adr/*.md]
    changes:
      sources: [CHANGELOG.md]
```

| Chave | Título padrão | Como aparece |
|---|---|---|
| `overview` | Visão geral | as fontes em sequência |
| `modules` | Módulos | um item por arquivo |
| `business_rules` | Regras de negócio | cada `##` é uma regra, com filtro |
| `data_model` | Modelo de dados | as fontes em sequência |
| `decisions` | Decisões | um item por arquivo |
| `security` | Segurança | as fontes em sequência |
| `delivery` | Ciclo de entrega | as fontes em sequência |
| `development` | Como manter e evoluir | as fontes em sequência |
| `operations` | Operação | as fontes em sequência |
| `out_of_scope` | Fora da versão | as fontes em sequência |
| `changes` | Novidades técnicas | os marcos do CHANGELOG com itens, inclusive o não publicado |

Seção sem `sources` não aparece.

Em `development`, conte como subir o ambiente e qual é o fluxo de
trabalho: branch, commits, revisão, pull request e CI. Conte também
como acrescentar uma funcionalidade, rodar os testes e publicar uma
versão.

## Diagramas

Um bloco de código `mermaid` vira diagrama. Use C4 de contexto e de
containers na visão geral e `erDiagram` no modelo de dados.

## Links

Links entre fontes do guia navegam dentro dele. Um link para outro
arquivo do repositório vai para `repository_url`.

## O guidebook não me deixa citar um arquivo

Sem `repository_url`, um link para arquivo que não é fonte do guia é
recusado. Defina `repository_url` ou ponha o arquivo numa seção.
