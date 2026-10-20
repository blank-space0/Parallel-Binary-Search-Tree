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
	DoGroups    bool
	DoCompare   bool
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

	if check["data-workers"] || check["comp-workers"] {
		args.DoGroups = true
	}
	if check["comp-workers"] {
		args.DoCompare = true
	}

	return args, nil
}
