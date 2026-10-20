# 0003. Limite padrão de 25 palavras por frase

Status: aceita (fase 1).

## Contexto

A regra de frase longa precisa de um padrão. Guias de linguagem simples
em português recomendam frases curtas, em geral entre 20 e 25 palavras.
Um limite baixo demais recusa frases técnicas legítimas no guia do
desenvolvedor.

## Decisão

Padrão de 25 palavras por frase, configurável por projeto e por guia
(`max_sentence_words`, de 5 a 200).

## Consequência

O guia do usuário fica perto do recomendado sem configuração. Projetos
com texto mais técnico sobem o limite só no guia do desenvolvedor.
