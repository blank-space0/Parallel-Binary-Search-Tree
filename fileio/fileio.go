package fileio

import (
	"os"
	"strconv"
	"strings"
)

func ReadFile(input string) ([][]int, error) {
	ret := [][]int{}
	dataBytes, err := os.ReadFile(input)
	if err != nil {
		return nil, err
	}
	dataStr := string(dataBytes)

	for line := range strings.Lines(dataStr) {
		curr := []int{}
		for elem := range strings.FieldsSeq(line) {
			num, err := strconv.Atoi(elem)
			if err != nil {
				return nil, err
			}
			curr = append(curr, num)
		}
		// empty line case, skip it to avoid overhead of constructing an empty bst
		if len(curr) == 0 {
			continue
		}
		ret = append(ret, curr)
	}
	return ret, nil
}
