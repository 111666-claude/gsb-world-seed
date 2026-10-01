package worldseed

import "testing"

func TestCurrentVersionIsAccepted(t *testing.T) {
	if _, err := NewHasher(7, SeedVersion); err != nil {
		t.Fatalf("当前版本应该被接受：%v", err)
	}
}

func TestValueIsPositiveForTypicalSeed(t *testing.T) {
	hasher, _ := NewHasher(7, SeedVersion)
	if hasher.Value(1, 2) == 0 {
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
	if SeedVersion != 3 {
		t.Fatal("种子版本常量被改了")
	}
}
