package checksum

import (
	"fmt"
	"testing"
)

func TestParityBitCheck(t *testing.T) {
	content := []byte("lorem ipsum dolor sit amet consectetur adipiscing elit sed do eiusmod tempor incididunt ut labore et dolore magna aliqua")
	checksum := ParityBitCheck(content)
	fmt.Println(string(checksum))
}
