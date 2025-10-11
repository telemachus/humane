package grouppool

import "sync"

var pool = sync.Pool{
	New: func() any {
		gs := make([]string, 0, 10)
		return &gs
	},
}

func New() *[]string {
	return pool.Get().(*[]string) //nolint:errcheck // This cannot panic.
}

func Free(gs *[]string) {
	*gs = (*gs)[:0]
	pool.Put(gs)
}
