// Package worldseed 是分块世界种子：按分块坐标确定性派生、改一块只重算一块、版本迁移可逆、校验和去重。
package worldseed

import "errors"

// SeedVersion 是当前支持的种子版本，ChunkSize 是分块边长。
const (
	SeedVersion = 3
	ChunkSize   = 16
)

// 支持迁移的版本闭区间：SeedVersion 及其相邻版本。
const (
	minVersion = SeedVersion - 1
	maxVersion = SeedVersion + 1
)

// ErrSeedVersion 表示种子版本不受支持。
var ErrSeedVersion = errors.New("worldseed: 不支持的种子版本")

// Hasher 是分块哈希器：派生值按 (seed, version, cx, cy) 缓存，内存 O(缓存分块数)。
type Hasher struct {
	seed       uint64
	version    int
	cache      map[[2]int]uint64
	recomputed int
}

// NewHasher 建哈希器，只接受 SeedVersion 及其相邻版本，其余返回 ErrSeedVersion。
func NewHasher(seed uint64, version int) (*Hasher, error) {
	if version < minVersion || version > maxVersion {
		return nil, ErrSeedVersion
	}
	return &Hasher{seed: seed, version: version, cache: map[[2]int]uint64{}}, nil
}

// derive 是纯函数：同一 (seed, version, cx, cy) 永远得到同一值，与调用顺序、调用次数无关。
func derive(seed uint64, version, cx, cy int) uint64 {
	value := seed*0x9E3779B97F4A7C15 + uint64(version)*0xC2B2AE3D27D4EB4F
	value ^= uint64(cx) * 0x165667B19E3779F9
	value ^= uint64(cy) * 0x27D4EB2F165667C5
	value ^= value >> 29
	value *= 0xBF58476D1CE4E5B9
	value ^= value >> 32
	return value
}

// ChunkHash 派生一个分块的值：只由种子、版本与分块坐标决定；命中缓存直接返回，
// 未命中才派生一次并计入 Recomputed，单次摊还 O(1)。
func (h *Hasher) ChunkHash(cx, cy int) uint64 {
	key := [2]int{cx, cy}
	if value, ok := h.cache[key]; ok {
		return value
	}
	value := derive(h.seed, h.version, cx, cy)
	h.cache[key] = value
	h.recomputed++
	return value
}

// Invalidate 只让这一个分块失效并立即就地重算它（无论之前是否已缓存），其余缓存原样保留，
// 因此 Recomputed 只增加 1，随后整批重算时其它分块全部命中缓存。
func (h *Hasher) Invalidate(cx, cy int) {
	h.cache[[2]int{cx, cy}] = derive(h.seed, h.version, cx, cy)
	h.recomputed++
}

// Migrate 迁移到相邻版本（v+1 或 v-1），双向可逆：v → v+1 → v 后同一分块的值复原。
// 派生值依赖版本，迁移时清空旧版本缓存。目标版本相同、不相邻或不受支持时返回 false 且状态不变。
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

// Checksum 是一组分块的校验和：同一分块无论出现多少次都只算一次，求和与顺序无关，
// 因此增量增删（加/减一个分块的值）与全量重算始终一致。
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
