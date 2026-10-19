package main

import (
	"fmt"
	"github.com/blank-space0/Parallel-Binary-Search-Tree/cli"
	"github.com/blank-space0/Parallel-Binary-Search-Tree/fileio"
	"github.com/blank-space0/Parallel-Binary-Search-Tree/logic"
	"os"
)

func main() {
	args, err := cli.Parse()

	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	//fmt.Println(args)

	nums, err := fileio.ReadFile(args.Input)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	// now, nums is a 2d array, each element is a seperate list of numbers that we convert into a bst
	// initialze a zeroe'd slice of node pointers
	roots := make([]*logic.Node, len(nums))
	for i, line := range nums {
		root, err := logic.BuildTree(line)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			continue
		}
		roots[i] = root
	}

	// because we compute hashes with %1000, we could have up to 1000 different hashes
	// trees is a hashmap with hashes as keys, and trees with the same hashes as values
	const MAXSIZE = 1000
	hashGroups := make(map[int][]int, MAXSIZE)
	for id, root := range roots {
		hash := logic.HashCompute(root)
		hashGroups[hash] = append(hashGroups[hash], id)
	}
}
