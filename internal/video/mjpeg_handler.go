package video

import (
	"fmt"
	"log"
	"net/http"
	"time"
)

const mjpegBoundary = "frame"

// MJPEGHandler serves an MJPEG stream from the FrameStore.
type MJPEGHandler struct {
	store *FrameStore
}

// NewMJPEGHandler creates a new MJPEGHandler.
func NewMJPEGHandler(store *FrameStore) *MJPEGHandler {
	return &MJPEGHandler{store: store}
}

// ServeHTTP implements http.Handler for MJPEG streaming.
func (h *MJPEGHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "multipart/x-mixed-replace; boundary="+mjpegBoundary)
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	var seq uint64
	for {
		select {
		case <-r.Context().Done():
			return
		default:
		}

		var frame []byte
		if seq == 0 {
			// First frame: get current if available, otherwise wait
			frame, seq = h.store.Get()
			if frame == nil {
				time.Sleep(100 * time.Millisecond)
				continue
			}
		} else {
			frame, seq = h.store.WaitNext(seq)
		}

		header := fmt.Sprintf("\r\n--%s\r\nContent-Type: image/jpeg\r\nContent-Length: %d\r\n\r\n", mjpegBoundary, len(frame))
		if _, err := w.Write([]byte(header)); err != nil {
			log.Printf("mjpeg: client disconnected: %v", err)
			return
		}
		if _, err := w.Write(frame); err != nil {
			log.Printf("mjpeg: write error: %v", err)
			return
		}

		flusher.Flush()
	}
}
