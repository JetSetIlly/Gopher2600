package rbits

import (
	"log"
)

// rather than make the bit pattern every time we can look up the pattern in
// the following slices
var bitPatterns8b []Register
var bitPatterns16b []Register

func init() {
	var err error
	bitPatterns8b = make([]Register, 256)
	for i := 0; i < 256; i++ {
		bitPatterns8b[i] = createBitPattern(i, 8)
		if err != nil {
			log.Fatalln(err)
		}
	}

	bitPatterns16b = make([]Register, 65536)
	for i := 0; i < 65536; i++ {
		bitPatterns16b[i] = createBitPattern(i, 16)
		if err != nil {
			log.Fatalln(err)
		}
	}
}

func createBitPattern(val int, bitlen int) Register {
	r := make(Register, bitlen)
	i := 0
	j := bitlen - 1
	for j >= 0 {
		bv := bitVals[j]
		if val/bv != 0 {
			r[i] = true
			val = val - bv
		}
		i++
		j--
	}
	return r
}
