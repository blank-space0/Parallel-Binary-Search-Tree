package main

import (
	"fmt"
	"github.com/blank-space0/Parallel-Binary-Search-Tree/cli"
	"github.com/blank-space0/Parallel-Binary-Search-Tree/fileio"
	"github.com/blank-space0/Parallel-Binary-Search-Tree/logic"
	"os"
	"time"
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
	// roots can be indexed via IDs (line numbers), since we insert in line order
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

	if args.Mode >= 2 {
		// timer starts after tree building as per instructions
		hashGroupStart := time.Now()

		for id, root := range roots {
			hash := logic.HashCompute(root)
			hashGroups[hash] = append(hashGroups[hash], id)
		}

		hashGroupTime := time.Since(hashGroupStart)
		fmt.Printf("hashGroupTime: %f\n", hashGroupTime.Seconds())
		for key, value := range hashGroups {
			// skip unique groups
			if len(value) == 1 {
				continue
			}

			fmt.Printf("%v:", key)
			for _, num := range value {
				fmt.Printf(" %v", num)
			}
			fmt.Println()
		}

	} else {
		hashes := make([]int, len(nums))
		hashStart := time.Now()
		for id, root := range roots {
			hash := logic.HashCompute(root)
			hashes[id] = hash
		}

		hashTime := time.Since(hashStart)
		fmt.Printf("hashTime: %f\n", hashTime.Seconds())
	}

}
