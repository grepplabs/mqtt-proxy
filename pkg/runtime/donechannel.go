package runtime

import "sync"

type DoneChannel struct {
	done chan struct{}
	once sync.Once
}

func (s *DoneChannel) Close() {
	s.once.Do(func() {
		close(s.done)
	})
}

func (s *DoneChannel) Done() <-chan struct{} {
	return s.done
}

func NewDoneChannel() *DoneChannel {
	return &DoneChannel{done: make(chan struct{})}
}
