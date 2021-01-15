package pool

type Pool struct {
	pos int
	buf []byte
}

const maxpoolsize = 1024 * 1024 * 50

//const maxpoolsize = 1024 * 128

func (pool *Pool) Get(size int) []byte {
	if maxpoolsize-pool.pos < size {
		pool.pos = 0
		if maxpoolsize >= size {
			pool.buf = make([]byte, maxpoolsize)
		} else {
			pool.buf = make([]byte, size+1024)
		}

	}
	b := pool.buf[pool.pos : pool.pos+size]
	pool.pos += size
	return b
}

func NewPool() *Pool {
	return &Pool{
		buf: make([]byte, maxpoolsize),
	}
}
