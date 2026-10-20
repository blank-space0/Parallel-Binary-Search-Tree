package main

import (
	"fmt"
	"github.com/blank-space0/Parallel-Binary-Search-Tree/cli"
	"github.com/blank-space0/Parallel-Binary-Search-Tree/fileio"
	"github.com/blank-space0/Parallel-Binary-Search-Tree/logic"
	"os"
	"sync"
	"time"
)

func main() {
	args, err := cli.Parse()

	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	//fmt.Println(args)

	// nums is our raw file input
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

	// check if we need to create the groupings
	if args.DoGroups {
		// timer starts after tree building as per instructions
		hashGroupStart := time.Now()

		// compute hashes and group ids of same hash together
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
		// just compute hashes, no groupings
		hashes := make([]int, len(nums))
		hashStart := time.Now()
		////////////////////////////////////////////////////
		// pick sequential or parallel options
		if args.HashWorkers == 1 {
			sequentialHash(roots, hashes)
		} else {
			perTreeHash(roots, hashes)
		}

		hashTime := time.Since(hashStart)
		fmt.Printf("hashTime: %f\n", hashTime.Seconds())
	}

	// check if we need to do comparisons
	if args.DoCompare {
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
			// flatten each group (convert it to an array via inorder traversal)
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

func sequentialHash(roots []*logic.Node, hashes []int) {
	for id, root := range roots {
		hashes[id] = logic.HashCompute(root)
	}
}

// execute each hash in parallel
// we can use waitgroups to act as a barrier so we finish these operations before continuing with the program
func perTreeHash(roots []*logic.Node, hashes []int) {
	wg := sync.WaitGroup{}
	for id, root := range roots {
		wg.Add(1)
		go func() {
			defer wg.Done()
			hashes[id] = logic.HashCompute(root)
		}()
	}
	wg.Wait()
}

func hashWorkerHash(roots []*logic.Node, hashes []int, workerCount int) {
	wg := sync.WaitGroup{}
	for worker := 0; worker < workerCount; worker++ {
		wg.Add(1)
		numTrees := len(roots)
		// fixed partitioning, split the work up as evenly as we can among workers
		start := worker * numTrees / workerCount
		end := (worker + 1) * numTrees / workerCount
		go func() {
			defer wg.Done()
			for ; start < end; start++ {
				hashes[start] = logic.HashCompute(roots[start])
			}
		}()
	}
	wg.Wait()
}
