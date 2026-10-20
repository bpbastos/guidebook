// Package media otimiza a mídia do projeto para a saída (docs/spec.md,
// seção 9.4): PNG vira PNG com paleta, exata quando cabe em 256 cores e
// quantizada quando não cabe (ADR 0015).
package media

import (
	"bytes"
	"crypto/sha256"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"slices"
	"sync"
)

// maxColors é o tamanho da paleta de um PNG.
const maxColors = 256

var pngSignature = []byte("\x89PNG\r\n\x1a\n")

// Optimize devolve o PNG otimizado ou, quando o resultado não fica menor
// ou a entrada não é PNG de 8 bits, a própria entrada.
func Optimize(data []byte) []byte {
	return optimize(data, false)
}

// optimize faz o trabalho de Optimize; dither liga o pontilhado de
// Floyd-Steinberg na quantização.
func optimize(data []byte, dither bool) []byte {
	if !bytes.HasPrefix(data, pngSignature) {
		return data
	}
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return data
	}
	switch img.(type) {
	case *image.NRGBA64, *image.RGBA64, *image.Gray16:
		// A paleta tem 8 bits por canal: não seria sem perda.
		return data
	}
	src := toNRGBA(img)
	keys := make([]uint32, len(src.Pix)/4)
	for i := range keys {
		keys[i] = pixel(src.Pix[4*i:])
	}
	distinct, counts := histogram(keys)
	out := image.NewPaletted(src.Rect, nil)
	var index []uint8 // índice na paleta de cada cor distinta
	if len(distinct) <= maxColors {
		out.Palette = make(color.Palette, len(distinct))
		index = make([]uint8, len(distinct))
		for i, k := range distinct {
			out.Palette[i], index[i] = keyColor(k), uint8(i)
		}
	} else {
		out.Palette = quantize(distinct, counts)
		if dither {
			draw.FloydSteinberg.Draw(out, src.Rect, src, src.Rect.Min)
		} else {
			index = nearest(out.Palette, distinct)
		}
	}
	if index != nil {
		for i, k := range keys {
			j, _ := slices.BinarySearch(distinct, k)
			out.Pix[i] = index[j]
		}
	}
	var b bytes.Buffer
	if err := (&png.Encoder{CompressionLevel: png.BestCompression}).Encode(&b, out); err != nil || b.Len() >= len(data) {
		return data
	}
	return b.Bytes()
}

// toNRGBA copia a imagem para NRGBA de 8 bits, sem perda para os tipos
// de 8 bits que o png.Decode devolve. Pixel totalmente transparente vira
// {0, 0, 0, 0}: a cor por baixo não aparece.
func toNRGBA(img image.Image) *image.NRGBA {
	b := img.Bounds()
	dst := image.NewNRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	if n, ok := img.(*image.NRGBA); ok && n.Stride == 4*b.Dx() && b.Min == (image.Point{}) {
		copy(dst.Pix, n.Pix)
	} else {
		for y := b.Min.Y; y < b.Max.Y; y++ {
			for x := b.Min.X; x < b.Max.X; x++ {
				dst.SetNRGBA(x-b.Min.X, y-b.Min.Y, color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA))
			}
		}
	}
	for i := 0; i < len(dst.Pix); i += 4 {
		if dst.Pix[i+3] == 0 {
			dst.Pix[i], dst.Pix[i+1], dst.Pix[i+2] = 0, 0, 0
		}
	}
	return dst
}

// pixel junta os quatro canais numa chave com o alfa no byte mais alto:
// em ordem de chave, as cores transparentes vêm primeiro, e o bloco de
// transparência do PNG fica curto.
func pixel(p []uint8) uint32 {
	return uint32(p[3])<<24 | uint32(p[0])<<16 | uint32(p[1])<<8 | uint32(p[2])
}

func keyColor(k uint32) color.NRGBA {
	return color.NRGBA{R: uint8(k >> 16), G: uint8(k >> 8), B: uint8(k), A: uint8(k >> 24)}
}

// histogram devolve as cores distintas, em ordem de chave, e quantos
// pixels usam cada uma.
func histogram(keys []uint32) ([]uint32, []uint64) {
	sorted := slices.Clone(keys)
	slices.Sort(sorted)
	var distinct []uint32
	var counts []uint64
	for _, k := range sorted {
		if n := len(distinct); n > 0 && distinct[n-1] == k {
			counts[n-1]++
			continue
		}
		distinct = append(distinct, k)
		counts = append(counts, 1)
	}
	return distinct, counts
}

// entry é uma cor distinta da imagem e quantos pixels a usam.
type entry struct {
	c [4]uint8 // R, G, B, A
	n uint64
}

// box é um grupo de cores vizinhas, entries[lo:hi], com as somas para
// calcular a média e o erro quadrático.
type box struct {
	lo, hi int
	n      uint64
	sum    [4]uint64
	sq     [4]uint64
	err    float64
}

func (b *box) measure(entries []entry) {
	b.n, b.sum, b.sq = 0, [4]uint64{}, [4]uint64{}
	for _, e := range entries[b.lo:b.hi] {
		b.n += e.n
		for ch, v := range e.c {
			b.sum[ch] += e.n * uint64(v)
			b.sq[ch] += e.n * uint64(v) * uint64(v)
		}
	}
	b.err = sse(b.n, b.sum, b.sq)
}

// sse é o erro quadrático de trocar cada cor do grupo pela média.
func sse(n uint64, sum, sq [4]uint64) float64 {
	if n == 0 {
		return 0
	}
	var e float64
	for ch := range sum {
		s := float64(sum[ch])
		e += float64(sq[ch]) - s*s/float64(n)
	}
	return e
}

// quantize escolhe até maxColors cores por median cut guiado pelo erro:
// divide sempre o grupo de maior erro quadrático, no canal e no ponto
// que mais reduzem o erro. Tudo em ordem fixa; mesma imagem, mesma
// paleta.
func quantize(distinct []uint32, counts []uint64) color.Palette {
	entries := make([]entry, len(distinct))
	for i, k := range distinct {
		c := keyColor(k)
		entries[i] = entry{c: [4]uint8{c.R, c.G, c.B, c.A}, n: counts[i]}
	}

	boxes := []*box{{lo: 0, hi: len(entries)}}
	boxes[0].measure(entries)
	scratch := make([]entry, len(entries))
	for len(boxes) < maxColors {
		best := -1
		for i, b := range boxes {
			if b.hi-b.lo > 1 && b.err > 0 && (best < 0 || b.err > boxes[best].err) {
				best = i
			}
		}
		if best < 0 {
			break
		}
		left, right := split(boxes[best], entries, scratch)
		boxes[best] = left
		boxes = append(boxes, right)
	}

	keysOut := make([]uint32, len(boxes))
	for i, b := range boxes {
		var c [4]uint8
		for ch := range c {
			c[ch] = uint8((b.sum[ch] + b.n/2) / b.n)
		}
		keysOut[i] = pixel([]uint8{c[0], c[1], c[2], c[3]})
	}
	slices.Sort(keysOut)
	keysOut = slices.Compact(keysOut)
	pal := make(color.Palette, len(keysOut))
	for i, k := range keysOut {
		pal[i] = keyColor(k)
	}
	return pal
}

// split divide o grupo no canal e no valor que deixam a menor soma de
// erros dos dois lados. As cores ficam reordenadas pelo canal escolhido,
// de forma estável.
func split(b *box, entries, scratch []entry) (*box, *box) {
	type bucket struct {
		n        uint64
		sum, sq  [4]uint64
		distinct int
	}
	bestErr, bestCh, bestV := -1.0, 0, 0
	for ch := range 4 {
		var buckets [256]bucket
		for _, e := range entries[b.lo:b.hi] {
			k := &buckets[e.c[ch]]
			k.n += e.n
			k.distinct++
			for c, v := range e.c {
				k.sum[c] += e.n * uint64(v)
				k.sq[c] += e.n * uint64(v) * uint64(v)
			}
		}
		var n uint64
		var sum, sq [4]uint64
		seen := 0
		for v := range 255 {
			k := buckets[v]
			n += k.n
			seen += k.distinct
			for c := range 4 {
				sum[c] += k.sum[c]
				sq[c] += k.sq[c]
			}
			if seen == 0 || seen == b.hi-b.lo {
				continue
			}
			var rs, rq [4]uint64
			for c := range 4 {
				rs[c], rq[c] = b.sum[c]-sum[c], b.sq[c]-sq[c]
			}
			if e := sse(n, sum, sq) + sse(b.n-n, rs, rq); bestErr < 0 || e < bestErr {
				bestErr, bestCh, bestV = e, ch, v
			}
		}
	}

	// Partição estável: primeiro as cores com canal <= bestV.
	part := scratch[:0]
	for _, e := range entries[b.lo:b.hi] {
		if int(e.c[bestCh]) <= bestV {
			part = append(part, e)
		}
	}
	mid := b.lo + len(part)
	for _, e := range entries[b.lo:b.hi] {
		if int(e.c[bestCh]) > bestV {
			part = append(part, e)
		}
	}
	copy(entries[b.lo:b.hi], part)
	left, right := &box{lo: b.lo, hi: mid}, &box{lo: mid, hi: b.hi}
	left.measure(entries)
	right.measure(entries)
	return left, right
}

// nearest devolve, para cada cor, o índice da cor mais próxima da
// paleta, pela distância quadrática nos quatro canais; no empate, o
// menor índice. A busca percorre a paleta em ordem de verde, a partir do
// verde da cor, e para quando só a diferença de verde já passa da melhor
// distância.
func nearest(pal color.Palette, colors []uint32) []uint8 {
	type cand struct {
		c   [4]int
		idx int
	}
	byG := make([]cand, len(pal))
	for i, c := range pal {
		n := c.(color.NRGBA)
		byG[i] = cand{[4]int{int(n.R), int(n.G), int(n.B), int(n.A)}, i}
	}
	slices.SortStableFunc(byG, func(a, b cand) int { return a.c[1] - b.c[1] })
	index := make([]uint8, len(colors))
	for i, k := range colors {
		n := keyColor(k)
		px := [4]int{int(n.R), int(n.G), int(n.B), int(n.A)}
		start, _ := slices.BinarySearchFunc(byG, px[1], func(c cand, g int) int { return c.c[1] - g })
		best, bestIdx := -1, 0
		try := func(c cand) bool {
			dg := c.c[1] - px[1]
			if best >= 0 && dg*dg > best {
				return false
			}
			d := 0
			for ch := range 4 {
				x := px[ch] - c.c[ch]
				d += x * x
			}
			if best < 0 || d < best || (d == best && c.idx < bestIdx) {
				best, bestIdx = d, c.idx
			}
			return true
		}
		for j := start; j < len(byG) && try(byG[j]); j++ {
		}
		for j := start - 1; j >= 0 && try(byG[j]); j-- {
		}
		index[i] = uint8(bestIdx)
	}
	return index
}

// Cache guarda o resultado de Optimize pelo SHA-256 da entrada, para o
// serve não otimizar de novo a imagem que não mudou.
type Cache struct {
	mu       sync.Mutex
	entries  map[[sha256.Size]byte]*cached
	computed int
}

type cached struct {
	once sync.Once
	out  []byte
	used bool
}

// NewCache devolve um cache vazio.
func NewCache() *Cache { return &Cache{entries: map[[sha256.Size]byte]*cached{}} }

// Optimize é o Optimize do pacote, com cache. Pode ser chamado de várias
// goroutines.
func (c *Cache) Optimize(data []byte) []byte {
	key := sha256.Sum256(data)
	c.mu.Lock()
	e := c.entries[key]
	if e == nil {
		e = &cached{}
		c.entries[key] = e
	}
	e.used = true
	c.mu.Unlock()
	e.once.Do(func() {
		e.out = Optimize(data)
		c.mu.Lock()
		c.computed++
		c.mu.Unlock()
	})
	return e.out
}

// Misses diz quantas vezes o cache precisou otimizar.
func (c *Cache) Misses() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.computed
}

// Sweep descarta o que não foi usado desde o Sweep anterior.
func (c *Cache) Sweep() {
	c.mu.Lock()
	defer c.mu.Unlock()
	for k, e := range c.entries {
		if !e.used {
			delete(c.entries, k)
		}
		e.used = false
	}
}
