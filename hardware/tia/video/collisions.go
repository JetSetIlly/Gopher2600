package video

type collisions struct {
	CXm0p  uint8
	CXm1p  uint8
	CXp0fb uint8
	CXp1fb uint8
	CXm0fb uint8
	CXm1fb uint8
	CXblpf uint8
	CXppmm uint8
}

func (coll *collisions) clear() {
	coll.CXm0p = 0
	coll.CXm1p = 0
	coll.CXp0fb = 0
	coll.CXp1fb = 0
	coll.CXm0fb = 0
	coll.CXm1fb = 0
	coll.CXblpf = 0
	coll.CXppmm = 0
}
