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

// Update takes ownership of the jpeg slice. Caller must not modify it after this call.
func (fs *FrameStore) Update(jpeg []byte) {
	fs.mu.Lock()
	fs.data = jpeg // ownership transfer, no copy
	fs.seq++
	fs.mu.Unlock()
	fs.cond.Broadcast()
}

// Get returns a shared reference to the current frame (read-only).
func (fs *FrameStore) Get() ([]byte, uint64) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()
	return fs.data, fs.seq
}

// WaitNext blocks until a frame with a sequence number > seq is available.
// Returns a shared reference (read-only).
func (fs *FrameStore) WaitNext(seq uint64) ([]byte, uint64) {
	fs.mu.RLock()
	for fs.seq <= seq {
		fs.cond.Wait()
	}
	d, s := fs.data, fs.seq
	fs.mu.RUnlock()
	return d, s
}
