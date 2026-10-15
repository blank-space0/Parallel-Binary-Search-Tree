// each subdir is a package
// packages work at the directory level
package cli

import (
	"errors"
	"flag"
)

type Args struct {
	HashWorkers int
	DataWorkers int
	CompWorkers int
	Input       string
	Mode        int // 1 = hash only, 2 = hash + groups, 3 = hash + groups + compare
}

func Parse() (Args, error) {
	args := Args{}

	flag.IntVar(&args.HashWorkers, "hash-workers", 1, "number of goroutines that compute BST hashes")
	flag.IntVar(&args.DataWorkers, "data-workers", 1, "number of goroutines that update the hash map")
	flag.IntVar(&args.CompWorkers, "comp-workers", 1, "number of goroutines that compare trees")
	flag.StringVar(&args.Input, "input", "", "path to the input file")

	flag.Parse()

	// terminate if we didnt get an input
	if args.Input == "" {
		return Args{}, errors.New("no input was given")
	}

	// check which args we visited
	check := map[string]bool{}
	flag.Visit(func(f *flag.Flag) {
		check[f.Name] = true
	})

	// checks for which args we passed in to set the mode
	if check["hash-workers"] && check["data-workers"] && check["comp-workers"] {
		args.Mode = 3
	} else if check["hash-workers"] && check["data-workers"] {
		args.Mode = 2
	} else {
		args.Mode = 1
	}

	return args, nil
}
