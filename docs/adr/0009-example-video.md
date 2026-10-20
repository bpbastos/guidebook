# 0009. Vídeo sintético do exemplo gerado uma vez com ffmpeg

Status: aceita (fase 8).

## Contexto

O projeto de exemplo precisa de um vídeo que toque na prévia. O
guidebook não decodifica vídeo, mas um arquivo falso mostraria erro no
player. Escrever um WebM válido à mão não é viável. O Bruno decidiu usar
uma vez o ffmpeg já instalado na máquina.

## Decisão

Vídeo e poster gerados uma vez com o ffmpeg 4.4.2 da máquina
(`ffmpeg version 4.4.2-0ubuntu0.22.04.1`). A fonte é o padrão de teste
`testsrc2`, sem áudio e sem dado de ninguém:

```sh
ffmpeg -f lavfi -i testsrc2=size=640x400:rate=15:duration=3 -an \
  -c:v libvpx-vp9 -b:v 200k -deadline good -row-mt 1 -pix_fmt yuv420p \
  -map_metadata -1 -fflags +bitexact -flags +bitexact register-order.webm
ffmpeg -f lavfi -i testsrc2=size=640x400:rate=15:duration=1 -frames:v 1 \
  -map_metadata -1 -fflags +bitexact -flags +bitexact register-order.png
ffprobe -v error -show_entries format=duration -of default=nw=1 register-order.webm
```

Resultado em `testdata/example/docs/user/media/videos/`:

- `register-order.webm`: VP9, 640×400, 3,000 s pelo ffprobe, 70.510
  bytes, SHA-256
  `7a8646eae8c742412791486856613371e1fd84ce18c6361f58a5edca4218591c`;
- `register-order.png`: 33.490 bytes, SHA-256
  `2d5655923b57615c299f08d0e1f9b5e532982cc975d485ff5d379f5ea4ffc29b`.

O `duration_seconds` do manifesto é `3.0`, igual ao do ffprobe.

## Consequência

O exemplo tem um vídeo que toca, sem ferramenta nova no fluxo. O ffmpeg
não entra no Makefile, na CI nem na imagem do guidebook; quem quiser
regenerar usa os comandos acima. Os bytes podem variar com outra versão
do ffmpeg; o teste não depende deles.
