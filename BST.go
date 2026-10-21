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

// no tuples in go, so I define a struct to send pairs of data through a channel
type pair struct {
	hash int
	id   int
}

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

		switch {
		case args.HashWorkers == 1 && args.DataWorkers == 1:
			// sequential: compute hashes and group ids of same hash together in the main thread
			for id, root := range roots {
				hash := logic.HashCompute(root)
				hashGroups[hash] = append(hashGroups[hash], id)
			}
		case args.HashWorkers > 1 && args.DataWorkers == 1:
			// hash workers send (hash, id) pairs over a channel to a single inserter
			parallelInsert(roots, hashGroups, args.HashWorkers)
		case args.HashWorkers > 1 && args.DataWorkers == args.HashWorkers:
			// each hash worker inserts into the map itself while holding a single mutex
			lockWorkersInsert(roots, hashGroups, args.HashWorkers)
		default:
			// other combinations (e.g. the optional i > j > 1 case) are not implemented
			fmt.Fprintf(os.Stderr, "unsupported combination: -hash-workers=%d -data-workers=%d\n", args.HashWorkers, args.DataWorkers)
			os.Exit(1)
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

		if args.HashWorkers == 1 {
			sequentialHash(roots, hashes)
		} else {
			hashWorkerHash(roots, hashes, args.HashWorkers)
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

// execute each hash in parallel over a thread pool
func hashWorkerHash(roots []*logic.Node, hashes []int, workerCount int) {
	workerWait := sync.WaitGroup{}
	for worker := 0; worker < workerCount; worker++ {
		workerWait.Add(1)
		numTrees := len(roots)
		// fixed partitioning, split the work up as evenly as we can among workers
		start := worker * numTrees / workerCount
		end := (worker + 1) * numTrees / workerCount
		go func() {
			defer workerWait.Done()
			for curr := start; curr < end; curr++ {
				hashes[curr] = logic.HashCompute(roots[curr])
			}
		}()
	}
	workerWait.Wait()
}

func parallelInsert(roots []*logic.Node, hashes map[int][]int, workerCount int) {
	workerWait := sync.WaitGroup{}
	insertWait := sync.WaitGroup{}
	// buffering is a slight optimization, now workers dont have to wait as much on the insertions
	ch := make(chan pair, workerCount)

	// spawn a single goroutine to do the map inserts
	insertWait.Add(1)
	go func() {
		defer insertWait.Done()
		insertIntoMap(hashes, ch)
	}()

	for worker := range workerCount {
		workerWait.Add(1)
		start := worker * len(roots) / workerCount
		end := (worker + 1) * len(roots) / workerCount
		go func() {
			defer workerWait.Done()
			for curr := start; curr < end; curr++ {
				hash := logic.HashCompute(roots[curr])
				ch <- pair{hash, curr}
			}
		}()
	}

	workerWait.Wait()
	close(ch)
	insertWait.Wait()
}

func insertIntoMap(hashes map[int][]int, ch chan pair) {
	for hashPair := range ch {
		hashes[hashPair.hash] = append(hashes[hashPair.hash], hashPair.id)
	}
}

func lockWorkersInsert(roots []*logic.Node, hashes map[int][]int, workerCount int) {
	lock := sync.Mutex{}
	workerWait := sync.WaitGroup{}

	for worker := range workerCount {
		workerWait.Add(1)
		start := worker * len(roots) / workerCount
		end := (worker + 1) * len(roots) / workerCount

		go func() {
			defer workerWait.Done()
			for curr := start; curr < end; curr++ {
				hash := logic.HashCompute(roots[curr])
				lock.Lock()
				hashes[hash] = append(hashes[hash], curr)
				lock.Unlock()
			}
		}()

	}
	workerWait.Wait()
}
