# Comandos

## build

```sh
guidebook build [-config guidebook.yaml] [-out out] [-guide all|user|developer]
```

Confere a documentação e grava os guias em `-out`.

O `build` otimiza as imagens PNG de mídia: capturas, imagens das páginas
e posters. Com até 256 cores, a imagem ganha uma paleta exata, sem
perda. Com mais cores, ela é reduzida a uma paleta de 256 cores. Se o
resultado não ficar menor, sai o arquivo original. JPEG, vídeos e o logo
saem como vieram. Caminho, nome e dimensões não mudam.

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
