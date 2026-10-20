package media

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"sync"
	"testing"
)

func encode(t *testing.T, img image.Image) []byte {
	t.Helper()
	var b bytes.Buffer
	// Sem compressão, para que a versão com paleta fique menor.
	if err := (&png.Encoder{CompressionLevel: png.NoCompression}).Encode(&b, img); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func decode(t *testing.T, data []byte) image.Image {
	t.Helper()
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	return img
}

// screen imita uma captura: fundo liso, faixas de cor e "texto" com
// antisserrilhado, em n cores distintas no total.
func screen(w, h, n int) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			i := (x/3 + y/5*7) % n
			img.SetNRGBA(x, y, color.NRGBA{uint8(i * 37), uint8(i * 11), uint8(255 - i*5), 255})
		}
	}
	return img
}

// gradient tem mais de 256 cores: cada pixel muda um pouco.
func gradient(w, h int) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			img.SetNRGBA(x, y, color.NRGBA{uint8(x), uint8(y), uint8((x + y) / 2), 255})
		}
	}
	return img
}

func samePixels(t *testing.T, want, got image.Image) {
	t.Helper()
	if want.Bounds() != got.Bounds() {
		t.Fatalf("dimensões mudaram: %v → %v", want.Bounds(), got.Bounds())
	}
	b := want.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			w := color.NRGBAModel.Convert(want.At(x, y)).(color.NRGBA)
			g := color.NRGBAModel.Convert(got.At(x, y)).(color.NRGBA)
			if w.A == 0 && g.A == 0 {
				continue
			}
			if w != g {
				t.Fatalf("pixel (%d,%d): %v → %v", x, y, w, g)
			}
		}
	}
}

func TestExactPalette(t *testing.T) {
	for _, n := range []int{2, 200, 256} {
		src := screen(320, 200, n)
		in := encode(t, src)
		out := Optimize(in)
		if len(out) >= len(in) {
			t.Fatalf("%d cores: %d bytes → %d; esperava menor", n, len(in), len(out))
		}
		got := decode(t, out)
		p, ok := got.(*image.Paletted)
		if !ok {
			t.Fatalf("%d cores: saiu %T; esperava PNG com paleta", n, got)
		}
		if len(p.Palette) != n {
			t.Errorf("%d cores: paleta com %d", n, len(p.Palette))
		}
		samePixels(t, src, got)
	}
}

func TestQuantize(t *testing.T) {
	src := gradient(256, 256)
	in := encode(t, src)
	out := Optimize(in)
	if len(out) >= len(in) {
		t.Fatalf("%d bytes → %d; esperava menor", len(in), len(out))
	}
	got := decode(t, out)
	p, ok := got.(*image.Paletted)
	if !ok {
		t.Fatalf("saiu %T; esperava PNG com paleta", got)
	}
	if len(p.Palette) > 256 {
		t.Fatalf("paleta com %d cores", len(p.Palette))
	}
	if got.Bounds() != src.Bounds() {
		t.Fatalf("dimensões mudaram: %v", got.Bounds())
	}
	// Erro médio por canal pequeno: a quantização segue a imagem.
	var sum, count float64
	for y := range 256 {
		for x := range 256 {
			w := src.NRGBAAt(x, y)
			g := color.NRGBAModel.Convert(got.At(x, y)).(color.NRGBA)
			for _, d := range []int{int(w.R) - int(g.R), int(w.G) - int(g.G), int(w.B) - int(g.B)} {
				sum += float64(d * d)
				count++
			}
		}
	}
	if mse := sum / count; mse > 30 {
		t.Errorf("erro quadrático médio %.1f por canal; esperava até 30", mse)
	}
}

func TestAlpha(t *testing.T) {
	// Poucas cores, com transparência total e parcial: paleta exata.
	src := screen(200, 100, 50)
	for y := range 100 {
		for x := range 40 {
			src.SetNRGBA(x, y, color.NRGBA{uint8(x), 0, 0, 0})
		}
		src.SetNRGBA(100, y, color.NRGBA{10, 20, 30, 128})
	}
	got := decode(t, Optimize(encode(t, src)))
	if _, ok := got.(*image.Paletted); !ok {
		t.Fatalf("saiu %T; esperava PNG com paleta", got)
	}
	samePixels(t, src, got)

	// Muitas cores com alfa: a transparência continua.
	grad := gradient(256, 256)
	for y := range 256 {
		for x := range 64 {
			c := grad.NRGBAAt(x, y)
			c.A = 0
			grad.SetNRGBA(x, y, c)
		}
	}
	got = decode(t, Optimize(encode(t, grad)))
	if _, ok := got.(*image.Paletted); !ok {
		t.Fatalf("saiu %T; esperava PNG com paleta", got)
	}
	for y := range 256 {
		if _, _, _, a := got.At(10, y).RGBA(); a != 0 {
			t.Fatalf("pixel (10,%d) deixou de ser transparente", y)
		}
		if _, _, _, a := got.At(200, y).RGBA(); a != 0xffff {
			t.Fatalf("pixel (200,%d) deixou de ser opaco", y)
		}
	}
}

func TestDeterministic(t *testing.T) {
	in := encode(t, gradient(300, 200))
	first := Optimize(in)
	var wg sync.WaitGroup
	results := make([][]byte, 8)
	for i := range results {
		wg.Go(func() { results[i] = Optimize(in) })
	}
	wg.Wait()
	for i, r := range results {
		if !bytes.Equal(r, first) {
			t.Fatalf("execução %d deu bytes diferentes", i)
		}
	}
}

func TestKeepsOriginal(t *testing.T) {
	// Já comprimido ao máximo, com paleta: otimizar não fica menor.
	small := image.NewPaletted(image.Rect(0, 0, 4, 4), color.Palette{color.NRGBA{A: 255}, color.NRGBA{255, 255, 255, 255}})
	small.SetColorIndex(1, 1, 1)
	var b bytes.Buffer
	(&png.Encoder{CompressionLevel: png.BestCompression}).Encode(&b, small)
	var jpg bytes.Buffer
	jpeg.Encode(&jpg, gradient(64, 64), nil)
	// PNG de 16 bits por canal sai como veio: a paleta perderia precisão.
	deep := image.NewNRGBA64(image.Rect(0, 0, 64, 64))
	deep.Set(1, 1, color.NRGBA64{0x1234, 0, 0, 0xffff})

	for name, in := range map[string][]byte{
		"PNG pequeno":  b.Bytes(),
		"JPEG":         jpg.Bytes(),
		"PNG 16 bits":  encode(t, deep),
		"não é imagem": []byte("texto qualquer"),
		"PNG quebrado": encode(t, gradient(8, 8))[:40],
	} {
		if out := Optimize(in); !bytes.Equal(out, in) {
			t.Errorf("%s: saída diferente da entrada", name)
		}
	}
}

func TestCache(t *testing.T) {
	c := NewCache()
	a := encode(t, screen(64, 64, 10))
	b := encode(t, gradient(64, 64))
	want := Optimize(a)
	for range 3 {
		if got := c.Optimize(a); !bytes.Equal(got, want) {
			t.Fatal("cache devolveu bytes diferentes")
		}
		c.Optimize(b)
	}
	if c.computed != 2 {
		t.Errorf("otimizou %d vezes; esperava 2, uma por entrada", c.computed)
	}

	// Sweep descarta só o que não foi usado desde o Sweep anterior.
	c.Sweep()
	c.Optimize(a)
	c.Sweep()
	c.Optimize(a)
	c.Optimize(b)
	if c.computed != 3 {
		t.Errorf("otimizou %d vezes; esperava 3 (b saiu do cache)", c.computed)
	}
}

// BenchmarkOptimize mede o pior caso: uma tela cheia em que quase todo
// pixel tem cor própria.
func BenchmarkOptimize(b *testing.B) {
	img := image.NewNRGBA(image.Rect(0, 0, 1920, 1080))
	seed := uint32(1)
	for i := range img.Pix {
		seed = seed*1664525 + 1013904223
		img.Pix[i] = uint8(seed >> 24)
		if i%4 == 3 {
			img.Pix[i] = 255
		}
	}
	var buf bytes.Buffer
	png.Encode(&buf, img)
	b.ResetTimer()
	for b.Loop() {
		Optimize(buf.Bytes())
	}
}

// TestNearest: a busca com poda dá o mesmo índice que a busca em toda a
// paleta, inclusive no empate.
func TestNearest(t *testing.T) {
	seed := uint32(7)
	next := func() uint8 {
		seed = seed*1664525 + 1013904223
		return uint8(seed >> 24)
	}
	pal := make(color.Palette, 256)
	for i := range pal {
		pal[i] = color.NRGBA{next() &^ 7, next() &^ 7, next() &^ 7, next() | 0xf0}
	}
	colors := make([]uint32, 5000)
	for i := range colors {
		colors[i] = pixel([]uint8{next(), next(), next(), next()})
	}
	got := nearest(pal, colors)
	for i, k := range colors {
		c := keyColor(k)
		best, want := -1, 0
		for j, p := range pal {
			n := p.(color.NRGBA)
			d := 0
			for _, x := range []int{int(c.R) - int(n.R), int(c.G) - int(n.G), int(c.B) - int(n.B), int(c.A) - int(n.A)} {
				d += x * x
			}
			if best < 0 || d < best {
				best, want = d, j
			}
		}
		if int(got[i]) != want {
			t.Fatalf("cor %v: índice %d; esperava %d", c, got[i], want)
		}
	}
}
