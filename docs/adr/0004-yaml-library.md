# 0004. Biblioteca YAML: go.yaml.in/yaml/v3

Status: aceita (fase 2).

## Contexto

A spec pede uma biblioteca YAML mantida. Em 2026-10-20, o
`gopkg.in/yaml.v3` está marcado como sem manutenção pelo autor (última
versão v3.0.1, de 2022). A organização YAML assumiu o código em
`github.com/yaml/go-yaml`, publicado como `go.yaml.in/yaml/v3` (v3.0.5,
de 2026-07-26) e `go.yaml.in/yaml/v4` (ainda release candidate,
v4.0.0-rc.6).

## Decisão

Usar `go.yaml.in/yaml/v3` v3.0.5, fixada no `go.mod`.

## Consequência

Mesma API do yaml.v3 conhecido, com manutenção ativa. Quando a v4 sair
estável, avaliamos a troca em ADR próprio.
