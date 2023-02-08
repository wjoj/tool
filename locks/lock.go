package locks

import "sync"

type Lock interface {
	Lock() error
	Unlock() error
}

type shareCall struct {
	wg    sync.WaitGroup
	err   error
	value any
}

type Share struct {
	calls map[string]*shareCall
	lock  sync.Mutex
}

func NewShare() *Share {
	return &Share{
		calls: make(map[string]*shareCall),
	}
}

func (s *Share) getLock(key string) (*shareCall, bool) {
	s.lock.Lock()
	call := s.calls[key]
	if call == nil {
		call = new(shareCall)
		call.wg.Add(1)
		s.calls[key] = call
		s.lock.Unlock()
		return call, false
	}
	s.lock.Unlock()
	call.wg.Wait()
	return call, true
}

func (s *Share) LockWait(key string, f func() (any, error)) (any, bool, error) {
	call, done := s.getLock(key)
	if done {
		return call.value, false, call.err
	}
	defer func() {
		s.lock.Lock()
		delete(s.calls, key)
		s.lock.Unlock()
		call.wg.Done()
	}()
	call.value, call.err = f()
	return call.value, true, call.err
}
