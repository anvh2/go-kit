package checksum

func ParityBitCheck(data []byte) byte {
	var parity byte
	for _, b := range data {
		parity ^= b
	}
	return parity
}
