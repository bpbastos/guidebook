# Servir os guias

O `guidebook serve` gera o guia do usuário e o serve por HTTP. Quando
a documentação muda, ele gera de novo sozinho.

O guia do desenvolvedor nunca é servido. Quem tem acesso ao repositório
gera esse guia com `guidebook build` e abre o arquivo no próprio
computador.

## Rode no seu computador

Na raiz do projeto, rode:

```sh
guidebook serve
```

Abra `http://127.0.0.1:8080`. O servidor confere a documentação a cada
2 segundos; mude com `-interval`.

## Ponha no compose do projeto

O serviço lê só o que o guia do usuário usa, somente leitura, e grava
só no `/tmp`, em memória. Ele fica na rede interna do compose; quem publica é
o proxy do projeto.

```yaml
services:
  guidebook:
    image: ghcr.io/bpbastos/guidebook:v0.1.1
    read_only: true
    tmpfs:
      - /tmp
    cap_drop: [ALL]
    security_opt:
      - no-new-privileges:true
    volumes:
      - ./guidebook.yaml:/project/guidebook.yaml:ro
      - ./docs/user:/project/docs/user:ro
      - ./CHANGELOG.md:/project/CHANGELOG.md:ro
    expose:
      - "8080"
```

As fontes do guia do desenvolvedor ficam fora do container. Se o logo
do produto não está em `docs/user/`, monte também a pasta dele.

Monte pastas sempre que puder. Um arquivo montado sozinho deixa de
mostrar mudanças quando o editor troca o arquivo por outro.

## Proteja o acesso

O guidebook não pede senha nem cuida de HTTPS. O proxy reverso do
projeto faz isso: autentica quem acessa e encaminha para
`http://guidebook:8080`. Não publique a porta 8080 direto.

## Desligue

`docker compose stop` manda SIGTERM. O guidebook espera as requisições
em curso, apaga os arquivos gerados e sai com código 0.

## Acompanhe pelo /healthz

| Resposta | Significado |
|---|---|
| 200 `ok` | o guia está em dia |
| 200 `stale` | o último build falhou; o guia anterior continua no ar |
| 503 `unavailable` | ainda não houve build bom |

Em `stale` e `unavailable`, o campo `problems` traz as recusas, no
mesmo formato do `check`.

## O guidebook não me deixa servir o guia

O `guidebook.yaml` não tem a seção `user`. O `serve` recusa com código
2, porque só serve o guia do usuário. Use `guidebook build` para gerar o
guia do desenvolvedor.

## O guidebook não me deixa ver a mudança

O build novo falhou. Veja `/healthz` ou o log do serviço, corrija e
salve de novo.
