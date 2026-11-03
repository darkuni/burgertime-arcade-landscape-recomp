package main

import (
	"fmt"
	"os"
	"sort"

	"burgertime/internal/assets"
	"burgertime/internal/sound"
)

func main() {
	ar, _ := assets.Load(os.Args[1])
	q := sound.New(ar.Sound[:])
	cost := map[string]map[int]int{}
	sound.OnNMI = func(p byte, c int) {
		k := fmt.Sprintf("phase%d", p)
		if cost[k] == nil {
			cost[k] = map[int]int{}
		}
		cost[k][c]++
	}
	q.Command(3)
	q.Run(3000000)
	var ks []string
	for k := range cost {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	for _, k := range ks {
		fmt.Println(k, cost[k])
	}
}
