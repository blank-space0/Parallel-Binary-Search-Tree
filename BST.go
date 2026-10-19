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

	if args.Mode == 3 {
		// initalize our comparison matrix
		matrix := make([][]bool, len(roots))
		for i := range matrix {
			matrix[i] = make([]bool, len(roots))
		}

		timeStart := time.Now()
		for _, groups := range hashGroups {
			// special case where we have groups of one (there will be no other tree to compare with so skip it)
			if len(groups) == 1 {
				matrix[groups[0]][groups[0]] = true
				continue
			}
			// flatten each group
			arrs := [][]int{}
			for _, id := range groups {
				arrs = append(arrs, logic.Flatten(roots[id], []int{}))
			}

			// compare each pair of arrays with same hashes
			for i := 0; i < len(arrs); i++ {
				for j := i + 1; j < len(arrs); j++ {
					isSame := logic.Compare(arrs[i], arrs[j])
					if isSame {
						matrix[groups[i]][groups[j]] = true
						matrix[groups[j]][groups[i]] = true
					}
				}
			}
		}

		compareTreeTime := time.Since(timeStart)
		fmt.Printf("compareTreeTime: %f\n", compareTreeTime.Seconds())

		// turn the matrix into groups of identical trees
		// each unvisited tree starts a group made of itself plus every tree it matches
		visited := make([]bool, len(roots))
		groupNum := 0
		for i := range matrix {
			if visited[i] {
				continue
			}

			visited[i] = true
			group := []int{i}
			for j := i + 1; j < len(matrix); j++ {
				if matrix[i][j] {
					visited[j] = true
					group = append(group, j)
				}
			}

			// only print groups with more than one tree
			if len(group) > 1 {
				fmt.Printf("group %d:", groupNum)
				for _, id := range group {
					fmt.Printf(" %d", id)
				}
				fmt.Println()
				groupNum++
			}
		}
	}

}
