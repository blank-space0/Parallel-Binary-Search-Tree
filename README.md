# Parallel BST Equivalence in Go

Finds which binary search trees are equivalent (same values in order) using parallel hashing and comparison.

Can be run sequentially, or with workers (goroutines).

## Building

```bash
go build -o BST .
```

## Running

Workers can be specified with 3 flags, and input with its own flag:

- -hash-workers=number of goroutines that hash the trees (int)
- -data-workers=number of goroutines that update the hash map (int)
- -comp-workers=number of goroutines that compare trees (int)
- -input=path to input file (string)

Which flags you pass decides what runs:

- -hash-workers only: just hashes the trees, prints hashTime
- -hash-workers and -data-workers: also groups trees by hash, prints hashGroupTime and the hash groups
- all three: also compares trees in each hash group, prints compareTreeTime and the groups of identical trees

Example:

```bash
./BST -hash-workers=1 -data-workers=1 -comp-workers=1 -input=inputs/simple.txt
```

Details of the assignment can be found here: <https://www.cs.utexas.edu/~rossbach/cs380p/lab/go-cs380p.html>
