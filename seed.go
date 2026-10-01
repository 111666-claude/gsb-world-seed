// Package worldseed 是世界种子：按坐标确定性派生、版本校验、集合校验和与顺序无关。
// 缺陷：把调用序号混进派生（同一坐标每次结果不同）、不校验种子版本、校验和把重复坐标重复计入。
package worldseed

import "errors"

// SeedVersion 是当前支持的种子版本。
const SeedVersion = 3

// Hasher 是按坐标派生的哈希器。
type Hasher struct {
	seed  uint64
	calls int
}

// NewHasher 建哈希器。缺陷：不校验版本，任何版本都接受。
func NewHasher(seed uint64, version int) (*Hasher, error) {
	return &Hasher{seed: seed}, nil
}

// Value 派生某个坐标的值。缺陷：把调用序号混进结果，同一坐标第二次就变了。
func (h *Hasher) Value(x, y int) uint64 {
	h.calls++
	value := h.seed*31 + uint64(x)*7 + uint64(y)*13 + uint64(h.calls)
	return value * 2654435761
}

// Checksum 是一组地块的校验和。缺陷：重复坐标会被重复计入。
func (h *Hasher) Checksum(cells [][2]int) uint64 {
	var sum uint64
	for _, cell := range cells {
		sum += h.Value(cell[0], cell[1])
	}
	return sum
}

// Calls 是累计派生次数（规模观测）。
func (h *Hasher) Calls() int { return h.calls }

// ErrSeedVersion 表示种子版本不受支持。
var ErrSeedVersion = errors.New("worldseed: 不支持的种子版本")
