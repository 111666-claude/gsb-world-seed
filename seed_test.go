package worldseed

import (
	"errors"
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

// ---------- README 口径补充用例 ----------

func TestNewHasherRejectsUnsupportedVersion(t *testing.T) {
	if _, err := NewHasher(7, SeedVersion); err != nil {
		t.Fatalf("当前版本应该被接受：%v", err)
	}
	for _, version := range []int{SeedVersion - 1, SeedVersion + 1} {
		if _, err := NewHasher(7, version); err != nil {
			t.Fatalf("相邻版本 %d 应该被接受：%v", version, err)
		}
	}
	for _, version := range []int{SeedVersion - 2, SeedVersion + 2, 0, -1} {
		if _, err := NewHasher(7, version); !errors.Is(err, ErrSeedVersion) {
			t.Fatalf("不支持的版本 %d 应该返回 ErrSeedVersion，得到 %v", version, err)
		}
	}
}

func TestChunkHashIsCoordinateDeterministic(t *testing.T) {
	a, _ := NewHasher(42, SeedVersion)
	b, _ := NewHasher(42, SeedVersion)
	// 两个独立哈希器、不同调用顺序、不同调用次数，同一坐标必须得到同值。
	a.ChunkHash(7, 7)
	a.ChunkHash(-3, 9)
	for i := 0; i < 100; i++ {
		a.ChunkHash(i, i)
	}
	b.ChunkHash(-3, 9)
	if got, want := a.ChunkHash(-3, 9), b.ChunkHash(-3, 9); got != want {
		t.Fatalf("同一坐标派生出不同值：%d != %d", got, want)
	}
	first := a.ChunkHash(5, 5)
	for i := 0; i < 100; i++ {
		a.ChunkHash(i, i)
	}
	if a.ChunkHash(5, 5) != first {
		t.Fatal("同一分块第三次派生与第一次不一致")
	}
	// 换种子或换版本后值应不同，换回来后复原。
	otherSeed, _ := NewHasher(43, SeedVersion)
	if otherSeed.ChunkHash(5, 5) == first {
		t.Fatal("不同种子不应派生出相同值")
	}
	c, _ := NewHasher(42, SeedVersion+1)
	if c.ChunkHash(5, 5) == first {
		t.Fatal("不同版本不应派生出相同值")
	}
}

func TestInvalidateOnlyRecomputesOneChunk(t *testing.T) {
	h, _ := NewHasher(7, SeedVersion)
	for i := 0; i < 1000; i++ {
		h.ChunkHash(i, 0)
	}
	before := h.Recomputed()
	// (5,5) 从未派生过，失效也只重算它一个。
	h.Invalidate(5, 5)
	if got := h.Recomputed(); got != before+1 {
		t.Fatalf("失效单个分块后 Recomputed 应只 +1：%d -> %d", before, got)
	}
	// 紧接着重算整批，全部命中缓存，重算次数不再增长；其它分块的值也保持不变。
	for i := 0; i < 1000; i++ {
		h.ChunkHash(i, 0)
	}
	if got := h.Recomputed(); got != before+1 {
		t.Fatalf("整批重算只有失效块需要重算，Recomputed 应保持 %d，实际 %d", before+1, got)
	}
	// 失效一个已缓存分块：只 +1，再派生时命中缓存且值不变。
	snapshot := h.Recomputed()
	value := h.ChunkHash(123, 0)
	h.Invalidate(123, 0)
	if h.Recomputed() != snapshot+1 {
		t.Fatal("失效已缓存分块也应只 +1")
	}
	if h.ChunkHash(123, 0) != value {
		t.Fatal("失效重算后确定性派生值应保持不变")
	}
	if h.Recomputed() != snapshot+1 {
		t.Fatal("重派生失效块应命中缓存，不应再计重算")
	}
}

func TestMigrateRoundtripRestoresValues(t *testing.T) {
	h, _ := NewHasher(7, SeedVersion)
	before := h.ChunkHash(1, 1)
	if !h.Migrate(SeedVersion + 1) {
		t.Fatal("向上迁移到相邻版本应该成功")
	}
	if h.Version() != SeedVersion+1 {
		t.Fatal("迁移后版本未更新")
	}
	if h.ChunkHash(1, 1) == before {
		t.Fatal("迁移到新版本后派生值应当变化")
	}
	if !h.Migrate(SeedVersion) {
		t.Fatal("迁回原版本应该成功")
	}
	if h.ChunkHash(1, 1) != before {
		t.Fatal("v -> v+1 -> v 后同一分块的值必须复原")
	}
	// 向下迁移同样双向可逆。
	down := h.ChunkHash(2, 2)
	if !h.Migrate(SeedVersion - 1) {
		t.Fatal("向下迁移到相邻版本应该成功")
	}
	if !h.Migrate(SeedVersion) {
		t.Fatal("从低版本迁回应该成功")
	}
	if h.ChunkHash(2, 2) != down {
		t.Fatal("v -> v-1 -> v 后同一分块的值必须复原")
	}
}

func TestMigrateRejectsNonAdjacentTargets(t *testing.T) {
	h, _ := NewHasher(7, SeedVersion)
	for _, target := range []int{SeedVersion, SeedVersion + 2, SeedVersion - 2, SeedVersion + 3} {
		if h.Migrate(target) {
			t.Fatalf("迁移到非相邻/相同版本 %d 应该返回 false", target)
		}
	}
	if h.Version() != SeedVersion {
		t.Fatal("被拒绝的迁移不应改变版本")
	}
}

func TestChecksumDeduplicatesIgnoresOrderAndMatchesIncremental(t *testing.T) {
	h, _ := NewHasher(7, SeedVersion)
	base := h.Checksum([][2]int{{1, 1}, {2, 2}, {3, 3}})
	if got := h.Checksum([][2]int{{3, 3}, {1, 1}, {2, 2}}); got != base {
		t.Fatalf("校验和与顺序有关：%d != %d", got, base)
	}
	if got := h.Checksum([][2]int{{1, 1}, {2, 2}, {3, 3}, {1, 1}, {2, 2}}); got != base {
		t.Fatalf("重复分块被重复计入：%d != %d", got, base)
	}
	single := h.ChunkHash(1, 1)
	if got := h.Checksum([][2]int{{1, 1}, {1, 1}}); got != single {
		t.Fatalf("两个相同分块的校验和应等于单个分块值：%d != %d", got, single)
	}
	// 增量增删与全量重算一致：全集 = 子集 + 新分块（uint64 回绕语义）。
	full := h.Checksum([][2]int{{1, 1}, {2, 2}, {3, 3}, {4, 4}})
	if got := base + h.ChunkHash(4, 4); got != full {
		t.Fatalf("增量加入与全量重算不一致：%d != %d", got, full)
	}
	if got := full - h.ChunkHash(4, 4); got != base {
		t.Fatalf("增量删除与全量重算不一致：%d != %d", got, base)
	}
}

// ---------- 参考模型对照 ----------

// reference 是 README 口径的独立参考实现：值永远直接由纯函数 derive 决定，
// 缓存只以「已知分块集合」建模用于统计重算次数；失效只重算一块；
// 迁移只接受相邻版本；校验和去重求和。
type reference struct {
	seed       uint64
	version    int
	known      map[[2]int]bool
	recomputed int
}

func newReference(seed uint64, version int) *reference {
	return &reference{seed: seed, version: version, known: map[[2]int]bool{}}
}

func (r *reference) chunkHash(cx, cy int) uint64 {
	key := [2]int{cx, cy}
	if !r.known[key] {
		r.known[key] = true
		r.recomputed++
	}
	return derive(r.seed, r.version, cx, cy)
}

func (r *reference) invalidate(cx, cy int) {
	r.known[[2]int{cx, cy}] = true
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
	r.known = map[[2]int]bool{}
	return true
}

func (r *reference) checksum(cells [][2]int) uint64 {
	seen := make(map[[2]int]struct{}, len(cells))
	var sum uint64
	for _, cell := range cells {
		if _, ok := seen[cell]; ok {
			continue
		}
		seen[cell] = struct{}{}
		sum += r.chunkHash(cell[0], cell[1])
	}
	return sum
}

func TestMatchesReferenceModel(t *testing.T) {
	rng := rand.New(rand.NewSource(20261002))
	for trial := 0; trial < 200; trial++ {
		seed := rng.Uint64()
		version := minVersion + rng.Intn(maxVersion-minVersion+1)
		h, err := NewHasher(seed, version)
		if err != nil {
			t.Fatalf("trial %d：建哈希器失败：%v", trial, err)
		}
		ref := newReference(seed, version)

		for step := 0; step < 40; step++ {
			// 随机「分块集合」：小坐标域制造缓存命中与重复分块。
			cells := make([][2]int, rng.Intn(12))
			for i := range cells {
				cells[i] = [2]int{rng.Intn(9) - 4, rng.Intn(9) - 4}
			}
			for _, cell := range cells {
				if got, want := h.ChunkHash(cell[0], cell[1]), ref.chunkHash(cell[0], cell[1]); got != want {
					t.Fatalf("trial %d step %d cell %v：派生值 %d != 参考 %d", trial, step, cell, got, want)
				}
			}

			// 随机「失效」。
			if len(cells) > 0 && rng.Intn(2) == 0 {
				cell := cells[rng.Intn(len(cells))]
				h.Invalidate(cell[0], cell[1])
				ref.invalidate(cell[0], cell[1])
			}

			// 随机「迁移」，目标覆盖相邻版本、相同版本、超界版本。
			if rng.Intn(3) == 0 {
				target := minVersion - 1 + rng.Intn(maxVersion-minVersion+3)
				if got, want := h.Migrate(target), ref.migrate(target); got != want {
					t.Fatalf("trial %d step %d Migrate(%d)：%v != 参考 %v", trial, step, target, got, want)
				}
			}

			// 校验和与参考一致；乱序并追加一份重复后仍然一致。
			if got, want := h.Checksum(cells), ref.checksum(cells); got != want {
				t.Fatalf("trial %d step %d：校验和 %d != 参考 %d", trial, step, got, want)
			}
			shuffled := append([][2]int(nil), cells...)
			rng.Shuffle(len(shuffled), func(i, j int) {
				shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
			})
			shuffled = append(shuffled, cells...)
			if got, want := h.Checksum(shuffled), ref.checksum(shuffled); got != want {
				t.Fatalf("trial %d step %d：乱序重复校验和 %d != 参考 %d", trial, step, got, want)
			}

			if got, want := h.Recomputed(), ref.recomputed; got != want {
				t.Fatalf("trial %d step %d：Recomputed %d != 参考 %d", trial, step, got, want)
			}
			if got, want := h.Version(), ref.version; got != want {
				t.Fatalf("trial %d step %d：Version %d != 参考 %d", trial, step, got, want)
			}
		}
	}
}

func BenchmarkChunkHash(b *testing.B) {
	h, _ := NewHasher(7, SeedVersion)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		h.ChunkHash(i%512, i/512%512)
	}
}
