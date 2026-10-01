# world-seed

世界种子：按坐标确定性派生、版本校验、集合校验和与顺序无关，只用 Go 标准库。

```
go test ./...
go run ./cmd/world-seed --sample stable
go run ./cmd/world-seed --sample version
go run ./cmd/world-seed --sample checksum
```

## 口径（README 为准）

- **坐标确定**：`Value(x, y)` 只由种子与坐标决定，与调用顺序、调用次数都无关。
- **版本校验**：`version` 不等于 `SeedVersion` 时返回 `ErrSeedVersion`。
- **集合校验和**：`Checksum` 与坐标顺序无关，同一个坐标出现多次只算一次。
- **不变量**：同一份种子与坐标集合重复计算结果相同；空集合校验和为 0。
- **规模**：单张地图 400 万格、每秒 20 万次派生，单次 O(1)，内存 O(1)。

## 输出契约（不改格式）

```
same=true
error=true
deduped=true
```

## 常量

```
SeedVersion = 3
```

## 目录

```
seed.go                 坐标派生与校验和
cmd/world-seed          命令行入口
seed_test.go            go test 用例
```
