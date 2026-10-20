# Comandos

## build

```sh
guidebook build [-config guidebook.yaml] [-out out] [-guide all|user|developer]
```

Confere a documentação e grava os guias em `-out`.

## check

```sh
guidebook check [-config guidebook.yaml] [-guide all|user|developer]
```

Faz as mesmas conferências do `build`, sem gravar nada.

## serve

```sh
guidebook serve [-config guidebook.yaml] [-addr 127.0.0.1:8080] [-interval 2s]
```

Gera o guia do usuário e o serve por HTTP, gerando de novo quando a
documentação muda. O guia do desenvolvedor nunca é servido. Veja [Servir os guias](../tasks/04-serve-guides.md).

## version

```sh
guidebook version
```

Mostra a versão do guidebook.

## Opções

| Opção | Padrão | O que faz |
|---|---|---|
| `-config` | `guidebook.yaml` | arquivo de configuração |
| `-out` | `out` | pasta de saída do `build` |
| `-guide` | `all` | guia a conferir ou gerar |
| `-addr` | `127.0.0.1:8080` | endereço do `serve` |
| `-interval` | `2s` | intervalo entre as conferências do `serve`, mínimo `500ms` |

## Códigos de saída

| Código | Quando |
|---|---|
| 0 | tudo certo |
| 1 | recusa de regra de escrita ou erro no conteúdo |
| 2 | uso errado, configuração inválida ou erro de leitura e gravação |
