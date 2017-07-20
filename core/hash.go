package core

func DBJHash(dat []byte) uint32 {
	hash := uint32(5381)
	for _, b := range dat {
		hash = (hash << 5) + hash + uint32(b)
	}
	return hash
}
