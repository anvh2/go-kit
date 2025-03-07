package sort

import (
	"fmt"
	"testing"
)

func TestQuickSort(t *testing.T) {
	arr := []int{5, 3, 2, 4, 1}
	QuickSort(arr)
	fmt.Println(arr)
}
