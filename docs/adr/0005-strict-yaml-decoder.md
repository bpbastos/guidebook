# 0005. Decodificação estrita própria sobre yaml.Node

Status: aceita (fase 2).

## Contexto

A spec pede validação do `guidebook.yaml` com erro em pt-BR, caminho do
campo e todos os erros de uma vez. O modo estrito do yaml.v3
(`KnownFields`) recusa chave desconhecida e chave repetida. Mas ele
responde em inglês, com o nome do tipo Go no lugar do caminho. E não
lista tudo: num teste com uma chave repetida e duas desconhecidas, só a
repetida apareceu.

## Decisão

O yaml.v3 só faz a análise sintática para `yaml.Node`. Um decodificador
pequeno, por reflexão sobre as tags `yaml` das structs, percorre a
árvore e anota cada problema com o caminho do campo
(`user.profiles[0].id`): chave desconhecida, chave repetida e tipo
errado. Os valores são validados depois, só se a estrutura estiver
boa.

## Consequência

Mensagens em pt-BR e com caminho, sem lista de campos duplicada à mão.
O decodificador aceita só os tipos que o `Config` usa: struct,
ponteiro, lista, mapa de texto, texto e inteiro. Tipo novo no `Config`
exige estendê-lo. Se isso for esquecido, o teste falha com pânico.
