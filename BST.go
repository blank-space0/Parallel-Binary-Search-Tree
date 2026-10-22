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

// a unit of comparison work, the ids of the two trees we want to compare
type idPair struct {
	idOne int
	idTwo int
}

// the spec only gives us -comp-workers, so there is no flag to pick between the two parallel versions
// flip this to true to run the goroutine per comparison version instead of the worker pool (for the report)
const goroutinePerCompare = false

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

		switch {
		case args.CompWorkers == 1:
			// sequential: do every comparison in the main thread
			sequentialCompare(roots, hashGroups, matrix)
		case args.CompWorkers > 1 && goroutinePerCompare:
			// spawn a goroutine for every single comparison
			goroutineCompare(roots, hashGroups, matrix)
		case args.CompWorkers > 1:
			// main thread pushes pairs into a bounded buffer, comp-workers goroutines pull them out
			bufferWorkersCompare(roots, hashGroups, matrix, args.CompWorkers)
		default:
			fmt.Fprintf(os.Stderr, "unsupported combination: -comp-workers=%d\n", args.CompWorkers)
			os.Exit(1)
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

// flatten every tree in the group (convert it to an array via inorder traversal)
// flats is indexed by tree id so workers can look up arrays from just an idPair
func flattenGroup(roots []*logic.Node, group []int, flats [][]int) {
	for _, id := range group {
		flats[id] = logic.Flatten(roots[id], []int{})
	}
}

// ids of every tree that shares its hash with another tree, these are the only ones we need to flatten
func multiTreeIds(hashGroups map[int][]int) []int {
	ids := []int{}
	for _, group := range hashGroups {
		if len(group) > 1 {
			ids = append(ids, group...)
		}
	}
	return ids
}

func sequentialCompare(roots []*logic.Node, hashGroups map[int][]int, matrix [][]bool) {
	flats := make([][]int, len(roots))
	for _, group := range hashGroups {
		// special case where we have groups of one (there will be no other tree to compare with so skip it)
		if len(group) == 1 {
			matrix[group[0]][group[0]] = true
			continue
		}
		flattenGroup(roots, group, flats)

		// compare each pair of arrays with same hashes
		for i := 0; i < len(group); i++ {
			for j := i + 1; j < len(group); j++ {
				one, two := group[i], group[j]
				if logic.Compare(flats[one], flats[two]) {
					matrix[one][two] = true
					matrix[two][one] = true
				}
			}
		}
	}
}

// first parallel version: one goroutine per comparison
// each goroutine writes to its own two cells of the matrix, so no lock is needed
func goroutineCompare(roots []*logic.Node, hashGroups map[int][]int, matrix [][]bool) {
	compareWait := sync.WaitGroup{}
	flats := make([][]int, len(roots))

	// flattening is most of the work (compare usually exits on the first mismatch)
	// so it has to be parallel too, same idea here: one goroutine per tree
	flattenWait := sync.WaitGroup{}
	for _, id := range multiTreeIds(hashGroups) {
		flattenWait.Add(1)
		go func() {
			defer flattenWait.Done()
			flats[id] = logic.Flatten(roots[id], []int{})
		}()
	}
	// every array has to be ready before any comparison reads it
	flattenWait.Wait()

	for _, group := range hashGroups {
		if len(group) == 1 {
			matrix[group[0]][group[0]] = true
			continue
		}

		for i := 0; i < len(group); i++ {
			for j := i + 1; j < len(group); j++ {
				one, two := group[i], group[j]
				compareWait.Add(1)
				go func() {
					defer compareWait.Done()
					if logic.Compare(flats[one], flats[two]) {
						matrix[one][two] = true
						matrix[two][one] = true
					}
				}()
			}
		}
	}
	compareWait.Wait()
}

// second parallel version: a fixed pool of comp-workers goroutines fed by a bounded buffer
func bufferWorkersCompare(roots []*logic.Node, hashGroups map[int][]int, matrix [][]bool, workerCount int) {
	workerWait := sync.WaitGroup{}
	flats := make([][]int, len(roots))
	// spec says the buffer can only hold up to comp-workers items at a time
	buffer := newCompBuffer(workerCount)

	// flattening is most of the work (compare usually exits on the first mismatch)
	// so split it across the workers first, same fixed partitioning as hashWorkerHash
	ids := multiTreeIds(hashGroups)
	flattenWait := sync.WaitGroup{}
	for worker := range workerCount {
		flattenWait.Add(1)
		start := worker * len(ids) / workerCount
		end := (worker + 1) * len(ids) / workerCount
		go func() {
			defer flattenWait.Done()
			for curr := start; curr < end; curr++ {
				flats[ids[curr]] = logic.Flatten(roots[ids[curr]], []int{})
			}
		}()
	}
	// every array has to be ready before any worker starts comparing
	flattenWait.Wait()

	for range workerCount {
		workerWait.Add(1)
		go func() {
			defer workerWait.Done()
			// keep pulling work until the buffer is closed and drained
			for {
				work, ok := buffer.pop()
				if !ok {
					return
				}
				if logic.Compare(flats[work.idOne], flats[work.idTwo]) {
					matrix[work.idOne][work.idTwo] = true
					matrix[work.idTwo][work.idOne] = true
				}
			}
		}()
	}

	// single thread (main) traverses the map and produces the work
	for _, group := range hashGroups {
		if len(group) == 1 {
			matrix[group[0]][group[0]] = true
			continue
		}

		for i := 0; i < len(group); i++ {
			for j := i + 1; j < len(group); j++ {
				buffer.push(idPair{group[i], group[j]})
			}
		}
	}

	// no more work, wake up any workers waiting on an empty buffer so they can exit
	buffer.close()
	workerWait.Wait()
}

// concurrent buffer: a ring buffer protected by one mutex
// notFull is waited on by the producer when the buffer is full
// notEmpty is waited on by the workers when the buffer is empty
// waiting on a condition sleeps the goroutine, so nobody spins
type compBuffer struct {
	lock     sync.Mutex
	notFull  *sync.Cond
	notEmpty *sync.Cond
	items    []idPair
	head     int
	count    int
	closed   bool
}

func newCompBuffer(capacity int) *compBuffer {
	buffer := &compBuffer{items: make([]idPair, capacity)}
	// both conditions share the buffer's mutex
	buffer.notFull = sync.NewCond(&buffer.lock)
	buffer.notEmpty = sync.NewCond(&buffer.lock)
	return buffer
}

func (buffer *compBuffer) push(work idPair) {
	buffer.lock.Lock()
	// loop instead of if, since we have to recheck the condition after waking up
	for buffer.count == len(buffer.items) {
		buffer.notFull.Wait()
	}
	// tail is head + count, wrapped around the end of the slice
	buffer.items[(buffer.head+buffer.count)%len(buffer.items)] = work
	buffer.count++
	buffer.notEmpty.Signal()
	buffer.lock.Unlock()
}

// returns false once the buffer is closed and there is nothing left to take
func (buffer *compBuffer) pop() (idPair, bool) {
	buffer.lock.Lock()
	for buffer.count == 0 && !buffer.closed {
		buffer.notEmpty.Wait()
	}
	if buffer.count == 0 {
		buffer.lock.Unlock()
		return idPair{}, false
	}
	work := buffer.items[buffer.head]
	buffer.head = (buffer.head + 1) % len(buffer.items)
	buffer.count--
	buffer.notFull.Signal()
	buffer.lock.Unlock()
	return work, true
}

func (buffer *compBuffer) close() {
	buffer.lock.Lock()
	buffer.closed = true
	// broadcast since every waiting worker needs to see that we are done
	buffer.notEmpty.Broadcast()
	buffer.lock.Unlock()
}
