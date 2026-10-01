// Package worldseed 是分块世界种子：按分块坐标确定性派生、改一块只重算一块、版本迁移可逆、校验和去重。
package worldseed

import "errors"

// SeedVersion 是当前支持的种子版本，ChunkSize 是分块边长。
const (
	SeedVersion = 3
	ChunkSize   = 16
)

// 支持的种子版本范围：从 1 到当前版本的下一版，供相邻版本之间双向迁移。
const (
	minVersion = 1
	maxVersion = SeedVersion + 1
)

// ErrSeedVersion 表示种子版本不受支持。
var ErrSeedVersion = errors.New("worldseed: 不支持的种子版本")

// Hasher 是分块哈希器。
type Hasher struct {
	seed       uint64
	version    int
	cache      map[[2]int]uint64
	recomputed int
}

// NewHasher 建哈希器；版本超出支持范围时返回 ErrSeedVersion。
func NewHasher(seed uint64, version int) (*Hasher, error) {
	if version < minVersion || version > maxVersion {
		return nil, ErrSeedVersion
	}
	return &Hasher{seed: seed, version: version, cache: map[[2]int]uint64{}}, nil
}

// derive 按坐标确定性派生：结果只由种子、版本与分块坐标决定，与调用顺序、调用次数无关。
func (h *Hasher) derive(cx, cy int) uint64 {
	value := h.seed*0x9E3779B97F4A7C15 + uint64(h.version)*0xC2B2AE3D27D4EB4F
	value ^= uint64(cx) * 0xBF58476D1CE4E5B9
	value ^= uint64(cy) * 0x94D049BB133111EB
	value ^= value >> 30
	value *= 0xBF58476D1CE4E5B9
	value ^= value >> 27
	value *= 0x94D049BB133111EB
	value ^= value >> 31
	return value
}

// ChunkHash 派生一个分块的值并缓存；缓存命中时直接返回，不产生重算。
func (h *Hasher) ChunkHash(cx, cy int) uint64 {
	key := [2]int{cx, cy}
	if value, ok := h.cache[key]; ok {
		return value
	}
	value := h.derive(cx, cy)
	h.cache[key] = value
	h.recomputed++
	return value
}

// Invalidate 只让这一个分块失效：立即重算它自己（Recomputed 只增加 1）并写回缓存，
// 其他分块的缓存原样保留，因此紧接着重算整批时只有它发生过这一次重算。
func (h *Hasher) Invalidate(cx, cy int) {
	h.cache[[2]int{cx, cy}] = h.derive(cx, cy)
	h.recomputed++
}

// Migrate 迁移到相邻版本（v-1 或 v+1），双向可逆：v → v+1 → v 后同一分块的值复原。
// 目标版本不相邻或超出支持范围时返回 false，且当前版本保持不变。
func (h *Hasher) Migrate(version int) bool {
	if version < minVersion || version > maxVersion {
		return false
	}
	if delta := version - h.version; delta != 1 && delta != -1 {
		return false
	}
	h.version = version
	h.cache = map[[2]int]uint64{}
	return true
}

// Checksum 是一组分块的校验和：同一分块出现多次只算一次；用加法聚合，与分块顺序无关。
func (h *Hasher) Checksum(cells [][2]int) uint64 {
	seen := make(map[[2]int]struct{}, len(cells))
	var sum uint64
	for _, cell := range cells {
		if _, ok := seen[cell]; ok {
			continue
		}
		seen[cell] = struct{}{}
		sum += h.ChunkHash(cell[0], cell[1])
	}
	return sum
}

// Recomputed 是累计重算次数（规模观测）。
func (h *Hasher) Recomputed() int { return h.recomputed }

// Version 是当前种子版本。
func (h *Hasher) Version() int { return h.version }
