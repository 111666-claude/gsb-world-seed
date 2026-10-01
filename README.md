# world-seed

分块世界种子：按分块坐标确定性派生、改一块只重算一块、版本迁移可逆、校验和去重，只用 Go 标准库。

```
go test ./...
go run ./cmd/world-seed --sample stable
go run ./cmd/world-seed --sample incremental
go run ./cmd/world-seed --sample migrate
```

## 口径（README 为准）

- **分块确定**：`ChunkHash(cx, cy)` 只由种子、版本与分块坐标决定，与调用顺序、调用次数无关，
  并且要缓存结果。
- **增量失效**：`Invalidate(cx, cy)` 只让这一个分块失效，紧接着重算整批分块时只有它需要重算，
  `Recomputed` 只增加 1。
- **版本迁移可逆**：`Migrate` 支持在 `SeedVersion` 与相邻版本之间双向迁移，
  `v → v+1 → v` 之后同一分块的值必须复原。
- **校验和**：与分块顺序无关，同一分块出现多次只算一次（增量增删后与全量重算一致）。
- **规模**：单张地图 400 万分块、每秒 20 万次派生，单次摊还 O(1)，内存 O(缓存分块数)。

## 输出契约（不改格式）

```
same=true
recomputed=1001
up=true roundtrip=true
deduped=true
```

## 常量

```
SeedVersion = 3
ChunkSize   = 16
```

## 目录

```
seed.go                 分块派生、失效与迁移
cmd/world-seed          命令行入口
seed_test.go            go test 用例
```
