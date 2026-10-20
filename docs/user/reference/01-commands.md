# Comandos

## build

```sh
guidebook build [-config guidebook.yaml] [-out out] [-guide all|user|developer]
                [-product-version <texto>] [-date AAAA-MM-DD]
```

Confere a documentação e grava os guias em `-out`.

O `build` otimiza as imagens PNG de mídia: capturas, imagens das páginas
e posters. Com até 256 cores, a imagem ganha uma paleta exata, sem
perda. Com mais cores, ela é reduzida a uma paleta de 256 cores. Se o
resultado não ficar menor, sai o arquivo original. JPEG, vídeos e o logo
saem como vieram. Caminho, nome e dimensões não mudam.

## check

```sh
guidebook check [-config guidebook.yaml] [-guide all|user|developer] [-no-media]
                [-product-version <texto>] [-date AAAA-MM-DD]
```

Com `-product-version` e `-date`, o rodapé dos dois guias mostra
"Versão 1.4.0 · Gerado em 20/10/2026". O guidebook nunca lê o relógio:
sem `-date`, não há data. Data inválida sai com código 2.

Faz as mesmas conferências do `build`, sem gravar nada no projeto. Para
medir os guias, gera tudo numa pasta temporária e apaga a pasta no fim.

Com `-no-media`, o `check` não lê nenhum arquivo de mídia. Serve para a
CI que roda antes do e2e, quando as capturas e os vídeos ainda não
existem. Ele continua conferindo as regras de escrita, o texto
alternativo, os links entre páginas e os caminhos das imagens. Não
confere marcadores, vídeos nem tamanho, e não gera os guias.

## Avisos

O `check` e o `build` avisam, na saída de erro, o que sobra na pasta
`media/` do guia do usuário:

```
aviso: docs/user/media/old.png: mídia sem uso
aviso: docs/user/media/old.markers.json: marcadores sem imagem usada
```

Aviso não muda o código de saída. Vídeo do manifesto fora de
`user.videos.tasks` também conta como sem uso. Apague o arquivo que
sobrou ou cite-o numa página.

## Tamanho dos guias

O `check` e o `build` informam, na saída padrão, o tamanho de cada guia:

```
build: guia do usuário com 9.876.543 bytes (9,4 MiB)
```

O número soma o HTML, os recursos que ele carrega e a mídia que ele
usa, já otimizada. Um arquivo usado pelos dois guias conta nos dois. O
`check` informa o mesmo número que o `build` gravaria.

Com `user.max_size_mib` ou `developer.max_size_mib`, um guia acima do
teto sai com código 1, com o tamanho e o limite. No `build`, os guias
ficam gravados mesmo assim, para você ver o que pesou.

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
| `-no-media` | desligada | só no `check`: não lê nem exige a mídia do e2e |
| `-product-version` | nenhuma | `build` e `check`: versão do produto no rodapé dos guias, até 40 caracteres |
| `-date` | nenhuma | `build` e `check`: data da geração no rodapé, no formato `AAAA-MM-DD` |
| `-addr` | `127.0.0.1:8080` | endereço do `serve` |
| `-interval` | `2s` | intervalo entre as conferências do `serve`, mínimo `500ms` |

## Códigos de saída

| Código | Quando |
|---|---|
| 0 | tudo certo |
| 1 | recusa de regra de escrita ou erro no conteúdo |
| 2 | uso errado, configuração inválida ou erro de leitura e gravação |
