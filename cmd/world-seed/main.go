// Command world-seed 跑分块种子样例。
package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"example.com/worldseed"
)

// Run 执行一次命令行调用，返回退出码。
func Run(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("world-seed", flag.ContinueOnError)
	flags.SetOutput(stderr)
	sample := flags.String("sample", "stable", "stable / incremental / migrate / checksum")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if *sample == "stable" {
		hasher, _ := worldseed.NewHasher(7, worldseed.SeedVersion)
		first := hasher.ChunkHash(1, 2)
		hasher.ChunkHash(9, 9)
		second := hasher.ChunkHash(1, 2)
		fmt.Fprintf(stdout, "same=%v\n", first == second)
		return 0
	}
	if *sample == "incremental" {
		hasher, _ := worldseed.NewHasher(7, worldseed.SeedVersion)
		for index := 0; index < 1000; index++ {
			hasher.ChunkHash(index, 0)
		}
		hasher.Invalidate(5, 5)
		for index := 0; index < 1000; index++ {
			hasher.ChunkHash(index, 0)
		}
		fmt.Fprintf(stdout, "recomputed=%d\n", hasher.Recomputed())
		return 0
	}
	if *sample == "migrate" {
		hasher, _ := worldseed.NewHasher(7, worldseed.SeedVersion)
		before := hasher.ChunkHash(1, 1)
		up := hasher.Migrate(worldseed.SeedVersion + 1)
		back := hasher.Migrate(worldseed.SeedVersion)
		fmt.Fprintf(stdout, "up=%v roundtrip=%v\n", up, back && hasher.ChunkHash(1, 1) == before)
		return 0
	}
	if *sample == "checksum" {
		hasher, _ := worldseed.NewHasher(7, worldseed.SeedVersion)
		single := hasher.ChunkHash(1, 1)
		again, _ := worldseed.NewHasher(7, worldseed.SeedVersion)
		deduped := again.Checksum([][2]int{{1, 1}, {1, 1}})
		fmt.Fprintf(stdout, "deduped=%v\n", deduped == single)
		return 0
	}
	fmt.Fprintln(stderr, "需要 --sample stable|incremental|migrate|checksum")
	return 2
}

func main() {
	os.Exit(Run(os.Args[1:], os.Stdout, os.Stderr))
}
