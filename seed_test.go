package worldseed

import (
	"math/rand"
	"testing"
)

func TestCurrentVersionIsAccepted(t *testing.T) {
	if _, err := NewHasher(7, SeedVersion); err != nil {
		t.Fatalf("当前版本应该被接受：%v", err)
	}
}

func TestChunkHashIsPositive(t *testing.T) {
	hasher, _ := NewHasher(7, SeedVersion)
	if hasher.ChunkHash(0, 0) == 0 {
		t.Fatal("派生值不该恒为 0")
	}
}

func TestChecksumOfEmptySet(t *testing.T) {
	hasher, _ := NewHasher(7, SeedVersion)
	if hasher.Checksum(nil) != 0 {
		t.Fatal("空集合的校验和应该是 0")
	}
}

func TestConstants(t *testing.T) {
	if SeedVersion != 3 || ChunkSize != 16 {
		t.Fatal("种子常量被改了")
	}
}

func TestChunkHashDeterministicByCoords(t *testing.T) {
	hasher, _ := NewHasher(7, SeedVersion)
	first := hasher.ChunkHash(3, 4)
	hasher.ChunkHash(100, 100)
	hasher.ChunkHash(-5, 9)
	if got := hasher.ChunkHash(3, 4); got != first {
		t.Fatalf("同一分块多次派生不一致：%d != %d", got, first)
	}
	fresh, _ := NewHasher(7, SeedVersion)
	if got := fresh.ChunkHash(3, 4); got != first {
		t.Fatalf("同一坐标换实例派生不一致：%d != %d", got, first)
	}
	otherSeed, _ := NewHasher(8, SeedVersion)
	if otherSeed.ChunkHash(3, 4) == first {
		t.Fatal("不同种子的派生值不该相同")
	}
	if hasher.ChunkHash(4, 3) == first {
		t.Fatal("不同坐标的派生值不该相同")
	}
	otherVersion, _ := NewHasher(7, SeedVersion+1)
	if otherVersion.ChunkHash(3, 4) == first {
		t.Fatal("不同版本的派生值不该相同")
	}
}

func TestInvalidateOnlyRecomputesOneChunk(t *testing.T) {
	hasher, _ := NewHasher(7, SeedVersion)
	const batch = 1000
	before := make([]uint64, batch)
	for index := 0; index < batch; index++ {
		before[index] = hasher.ChunkHash(index, 0)
	}
	if got := hasher.Recomputed(); got != batch {
		t.Fatalf("首批重算次数 = %d，期望 %d", got, batch)
	}
	hasher.Invalidate(5, 0)
	if got := hasher.Recomputed(); got != batch+1 {
		t.Fatalf("单块失效后重算次数 = %d，期望 %d", got, batch+1)
	}
	for index := 0; index < batch; index++ {
		if got := hasher.ChunkHash(index, 0); got != before[index] {
			t.Fatalf("分块 (%d, 0) 失效后值变了：%d != %d", index, got, before[index])
		}
	}
	if got := hasher.Recomputed(); got != batch+1 {
		t.Fatalf("整批重算后重算次数 = %d，期望只增加 1 到 %d", got, batch+1)
	}
}

func TestInvalidateOutsideBatch(t *testing.T) {
	hasher, _ := NewHasher(7, SeedVersion)
	const batch = 1000
	for index := 0; index < batch; index++ {
		hasher.ChunkHash(index, 0)
	}
	hasher.Invalidate(5, 5)
	for index := 0; index < batch; index++ {
		hasher.ChunkHash(index, 0)
	}
	if got := hasher.Recomputed(); got != batch+1 {
		t.Fatalf("批次外单块失效后重算次数 = %d，期望 %d", got, batch+1)
	}
}

func TestMigrateRoundtrip(t *testing.T) {
	hasher, _ := NewHasher(7, SeedVersion)
	coords := [][2]int{{0, 0}, {1, 1}, {-3, 7}, {15, 15}}
	before := make([]uint64, len(coords))
	for index, cell := range coords {
		before[index] = hasher.ChunkHash(cell[0], cell[1])
	}
	if !hasher.Migrate(SeedVersion + 1) {
		t.Fatal("迁移到下一版应该成功")
	}
	if hasher.Version() != SeedVersion+1 {
		t.Fatalf("迁移后版本 = %d，期望 %d", hasher.Version(), SeedVersion+1)
	}
	changed := 0
	for index, cell := range coords {
		if hasher.ChunkHash(cell[0], cell[1]) != before[index] {
			changed++
		}
	}
	if changed == 0 {
		t.Fatal("迁移后派生值应该随版本变化")
	}
	if !hasher.Migrate(SeedVersion) {
		t.Fatal("迁回当前版应该成功")
	}
	for index, cell := range coords {
		if got := hasher.ChunkHash(cell[0], cell[1]); got != before[index] {
			t.Fatalf("迁移往返后分块 %v 的值未复原：%d != %d", cell, got, before[index])
		}
	}
}

func TestMigrateRejectsNonAdjacent(t *testing.T) {
	hasher, _ := NewHasher(7, SeedVersion)
	for _, target := range []int{SeedVersion, SeedVersion - 2, SeedVersion + 2, 0, -1, maxVersion + 1} {
		if hasher.Migrate(target) {
			t.Fatalf("迁移到非相邻或非法版本 %d 应该失败", target)
		}
		if hasher.Version() != SeedVersion {
			t.Fatalf("迁移失败后版本不该变，得到 %d", hasher.Version())
		}
	}
	if _, err := NewHasher(7, 0); err != ErrSeedVersion {
		t.Fatalf("版本 0 应该报 ErrSeedVersion，得到 %v", err)
	}
	if _, err := NewHasher(7, maxVersion+1); err != ErrSeedVersion {
		t.Fatalf("版本 %d 应该报 ErrSeedVersion，得到 %v", maxVersion+1, err)
	}
	if _, err := NewHasher(7, maxVersion); err != nil {
		t.Fatalf("下一版应该可以直接建：%v", err)
	}
}

func TestChecksumDedupAndOrder(t *testing.T) {
	hasher, _ := NewHasher(7, SeedVersion)
	cells := [][2]int{{1, 1}, {2, 2}, {3, 3}}
	full := hasher.Checksum(cells)
	if got := hasher.Checksum([][2]int{{1, 1}, {2, 2}, {3, 3}, {1, 1}, {2, 2}}); got != full {
		t.Fatalf("重复分块不该重复计入：%d != %d", got, full)
	}
	if got := hasher.Checksum([][2]int{{3, 3}, {1, 1}, {2, 2}}); got != full {
		t.Fatalf("校验和应与顺序无关：%d != %d", got, full)
	}
	pair := hasher.Checksum([][2]int{{1, 1}, {2, 2}})
	if got := pair + hasher.ChunkHash(3, 3); got != full {
		t.Fatalf("增量增加后与全量重算不一致：%d != %d", got, full)
	}
	if got := full - hasher.ChunkHash(3, 3); got != pair {
		t.Fatalf("增量删除后与全量重算不一致：%d != %d", got, pair)
	}
}

func TestScaleFourMillionChunks(t *testing.T) {
	hasher, _ := NewHasher(7, SeedVersion)
	const total = 1 << 22 // 4,194,304，覆盖 400 万分块规模
	for index := 0; index < total; index++ {
		hasher.ChunkHash(index%2048, index/2048)
	}
	if got := hasher.Recomputed(); got != total {
		t.Fatalf("全量派生重算次数 = %d，期望 %d", got, total)
	}
	for index := 0; index < total; index++ {
		hasher.ChunkHash(index%2048, index/2048)
	}
	if got := hasher.Recomputed(); got != total {
		t.Fatalf("缓存命中后重算次数 = %d，期望仍为 %d", got, total)
	}
}

// reference 是口径的朴素参考实现：不缓存、每次纯派生，显式模拟单块失效与重算计数。
type reference struct {
	seed       uint64
	version    int
	fresh      map[[2]int]bool
	recomputed int
}

func refDerive(seed uint64, version, cx, cy int) uint64 {
	value := seed*0x9E3779B97F4A7C15 + uint64(version)*0xC2B2AE3D27D4EB4F
	value ^= uint64(cx) * 0xBF58476D1CE4E5B9
	value ^= uint64(cy) * 0x94D049BB133111EB
	value ^= value >> 30
	value *= 0xBF58476D1CE4E5B9
	value ^= value >> 27
	value *= 0x94D049BB133111EB
	value ^= value >> 31
	return value
}

func newReference(seed uint64, version int) *reference {
	return &reference{seed: seed, version: version, fresh: map[[2]int]bool{}}
}

func (r *reference) chunkHash(cx, cy int) uint64 {
	key := [2]int{cx, cy}
	if !r.fresh[key] {
		r.fresh[key] = true
		r.recomputed++
	}
	return refDerive(r.seed, r.version, cx, cy)
}

func (r *reference) invalidate(cx, cy int) {
	r.fresh[[2]int{cx, cy}] = true
	r.recomputed++
}

func (r *reference) migrate(version int) bool {
	if version < minVersion || version > maxVersion {
		return false
	}
	if delta := version - r.version; delta != 1 && delta != -1 {
		return false
	}
	r.version = version
	r.fresh = map[[2]int]bool{}
	return true
}

func (r *reference) checksum(cells [][2]int) uint64 {
	seen := make(map[[2]int]bool, len(cells))
	var sum uint64
	for _, cell := range cells {
		if seen[cell] {
			continue
		}
		seen[cell] = true
		sum += r.chunkHash(cell[0], cell[1])
	}
	return sum
}

func TestDifferentialAgainstReference(t *testing.T) {
	rng := rand.New(rand.NewSource(20261002))
	for trial := 0; trial < 200; trial++ {
		seed := rng.Uint64()
		version := minVersion + rng.Intn(maxVersion-minVersion+1)
		hasher, err := NewHasher(seed, version)
		if err != nil {
			t.Fatalf("第 %d 组：建哈希器失败：%v", trial, err)
		}
		ref := newReference(seed, version)
		setSize := 1 + rng.Intn(64)
		cells := make([][2]int, setSize)
		for index := range cells {
			cells[index] = [2]int{rng.Intn(33) - 16, rng.Intn(33) - 16}
		}
		steps := 1 + rng.Intn(40)
		for step := 0; step < steps; step++ {
			switch rng.Intn(4) {
			case 0:
				for _, cell := range cells {
					if got, want := hasher.ChunkHash(cell[0], cell[1]), ref.chunkHash(cell[0], cell[1]); got != want {
						t.Fatalf("第 %d 组第 %d 步：分块 %v 派生值 %d != 参考 %d", trial, step, cell, got, want)
					}
				}
			case 1:
				cx, cy := rng.Intn(33)-16, rng.Intn(33)-16
				hasher.Invalidate(cx, cy)
				ref.invalidate(cx, cy)
			case 2:
				current := hasher.Version()
				targets := []int{current - 1, current + 1, current, current + 2, minVersion - 1, maxVersion + 1}
				target := targets[rng.Intn(len(targets))]
				if got, want := hasher.Migrate(target), ref.migrate(target); got != want {
					t.Fatalf("第 %d 组第 %d 步：迁移到 %d 返回值 %v != 参考 %v", trial, step, target, got, want)
				}
			case 3:
				count := rng.Intn(2*len(cells) + 1)
				sub := make([][2]int, count)
				for index := range sub {
					sub[index] = cells[rng.Intn(len(cells))]
				}
				if got, want := hasher.Checksum(sub), ref.checksum(sub); got != want {
					t.Fatalf("第 %d 组第 %d 步：校验和 %d != 参考 %d", trial, step, got, want)
				}
			}
			if got, want := hasher.Recomputed(), ref.recomputed; got != want {
				t.Fatalf("第 %d 组第 %d 步：重算次数 %d != 参考 %d", trial, step, got, want)
			}
			if got, want := hasher.Version(), ref.version; got != want {
				t.Fatalf("第 %d 组第 %d 步：版本 %d != 参考 %d", trial, step, got, want)
			}
		}
	}
}

func BenchmarkChunkHash(b *testing.B) {
	hasher, _ := NewHasher(7, SeedVersion)
	for index := 0; index < b.N; index++ {
		hasher.ChunkHash(index, 0)
	}
}
