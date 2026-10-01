// Command world-seed 跑世界种子样例。
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
	sample := flags.String("sample", "stable", "stable / version / checksum")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if *sample == "stable" {
		hasher, _ := worldseed.NewHasher(7, worldseed.SeedVersion)
		first := hasher.Value(1, 2)
		hasher.Value(9, 9)
		second := hasher.Value(1, 2)
		fmt.Fprintf(stdout, "same=%v\n", first == second)
		return 0
	}
	if *sample == "version" {
		_, err := worldseed.NewHasher(7, 99)
		fmt.Fprintf(stdout, "error=%v\n", err != nil)
		return 0
	}
	if *sample == "checksum" {
		hasher, _ := worldseed.NewHasher(7, worldseed.SeedVersion)
		single := hasher.Value(1, 1)
		again, _ := worldseed.NewHasher(7, worldseed.SeedVersion)
		deduped := again.Checksum([][2]int{{1, 1}, {1, 1}})
		fmt.Fprintf(stdout, "deduped=%v\n", deduped == single)
		return 0
	}
	fmt.Fprintln(stderr, "需要 --sample stable|version|checksum")
	return 2
}

func main() {
	os.Exit(Run(os.Args[1:], os.Stdout, os.Stderr))
}
