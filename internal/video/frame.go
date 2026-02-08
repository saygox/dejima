package video

import "sync"

// FrameStore holds the latest JPEG frame with thread-safe access.
type FrameStore struct {
	mu   sync.RWMutex
	data []byte
	seq  uint64
	cond *sync.Cond
}

// NewFrameStore creates a new FrameStore.
func NewFrameStore() *FrameStore {
	fs := &FrameStore{}
	fs.cond = sync.NewCond(fs.mu.RLocker())
	return fs
}

// Update replaces the stored frame with new JPEG data.
func (fs *FrameStore) Update(jpeg []byte) {
	fs.mu.Lock()
	fs.data = make([]byte, len(jpeg))
	copy(fs.data, jpeg)
	fs.seq++
	fs.mu.Unlock()
	fs.cond.Broadcast()
}

// Get returns a copy of the current frame data and its sequence number.
func (fs *FrameStore) Get() ([]byte, uint64) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()
	if fs.data == nil {
		return nil, 0
	}
	out := make([]byte, len(fs.data))
	copy(out, fs.data)
	return out, fs.seq
}

// WaitNext blocks until a frame with a sequence number > seq is available.
func (fs *FrameStore) WaitNext(seq uint64) ([]byte, uint64) {
	fs.mu.RLock()
	for fs.seq <= seq {
		fs.cond.Wait()
	}
	out := make([]byte, len(fs.data))
	copy(out, fs.data)
	s := fs.seq
	fs.mu.RUnlock()
	return out, s
}
