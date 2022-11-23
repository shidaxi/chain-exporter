package main

import (
	"fmt"
	"math/big"
	"testing"
)

func Test0(t *testing.T) {
	a := []interface{}{"a", 1, true}
	fmt.Println(a)
	a = append(a, 3.14)
	for _, v := range a {
		fmt.Println(v)
	}
}

func Test1(t *testing.T) {
	n := new(big.Int)
	n.SetString("0000000000000000000002", 10)
	fmt.Println(n)
}
