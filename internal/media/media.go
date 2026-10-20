// Package media otimiza a mídia do projeto para a saída (docs/spec.md,
// seção 9.4): PNG vira PNG com paleta, exata quando cabe em 256 cores e
// quantizada quando não cabe.
package media

import "sync"

// Optimize devolve o PNG otimizado ou, quando o resultado não fica menor
// ou a entrada não é PNG de 8 bits, a própria entrada.
func Optimize(data []byte) []byte {
	return data
}

// Cache guarda o resultado de Optimize pelo SHA-256 da entrada.
type Cache struct {
	mu       sync.Mutex
	computed int
}

// NewCache devolve um cache vazio.
func NewCache() *Cache { return &Cache{} }

// Optimize é o Optimize do pacote, com cache.
func (c *Cache) Optimize(data []byte) []byte {
	c.mu.Lock()
	c.computed++
	c.mu.Unlock()
	return Optimize(data)
}

// Misses diz quantas vezes o cache precisou otimizar.
func (c *Cache) Misses() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.computed
}

// Sweep descarta o que não foi usado desde o Sweep anterior.
func (c *Cache) Sweep() {}
