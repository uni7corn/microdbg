package encoding

type structSize []int

func (ss structSize) Add(size structSize) structSize {
	return append(ss, size...)
}

func (ss structSize) Size() (total int) {
	for _, size := range ss {
		total += size
	}
	return
}
