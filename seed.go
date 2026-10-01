// Package worldseed 是分块世界种子：按分块坐标确定性派生、改一块只重算一块、版本迁移可逆、校验和去重。
// 缺陷：把调用序号混进派生（同一坐标结果不同）、改一块就把整张缓存清掉（全图重算）、
// 版本迁移没有实现、校验和把重复分块重复计入。
package worldseed

import "errors"

// SeedVersion 是当前支持的种子版本，ChunkSize 是分块边长。
const (
	SeedVersion = 3
	ChunkSize   = 16
)

// ErrSeedVersion 表示种子版本不受支持。
var ErrSeedVersion = errors.New("worldseed: 不支持的种子版本")

// Hasher 是分块哈希器。
type Hasher struct {
	seed       uint64
	version    int
	cache      map[[2]int]uint64
	calls      int
	recomputed int
}

// NewHasher 建哈希器。缺陷：不校验版本，任何版本都接受。
func NewHasher(seed uint64, version int) (*Hasher, error) {
	return &Hasher{seed: seed, version: version, cache: map[[2]int]uint64{}}, nil
}

// ChunkHash 派生一个分块的值。缺陷：把调用序号混进结果（同一分块每次都不一样），也不写缓存。
func (h *Hasher) ChunkHash(cx, cy int) uint64 {
	h.calls++
	h.recomputed++
	value := h.seed*31 + uint64(cx)*7 + uint64(cy)*13 + uint64(h.calls)
	return value * 2654435761
}

// Invalidate 让某个分块失效。缺陷：直接把整张缓存清掉（其他分块被迫全图重算）。
func (h *Hasher) Invalidate(cx, cy int) {
	h.cache = map[[2]int]uint64{}
}

// Migrate 迁移种子版本。缺陷：没有实现。
func (h *Hasher) Migrate(version int) bool { return false }

// Checksum 是一组分块的校验和。缺陷：重复分块会被重复计入。
func (h *Hasher) Checksum(cells [][2]int) uint64 {
	var sum uint64
	for _, cell := range cells {
		sum += h.ChunkHash(cell[0], cell[1])
	}
	return sum
}

// Recomputed 是累计重算次数（规模观测）。
func (h *Hasher) Recomputed() int { return h.recomputed }

// Version 是当前种子版本。
func (h *Hasher) Version() int { return h.version }
