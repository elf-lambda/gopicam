package main

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"log"
	"net/http"
	"time"
)

const (
	WIDTH  = 1280
	HEIGHT = 720
	FPS    = 30
)

func generateDummyFrame(frameNum int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, WIDTH, HEIGHT))

	// Create a gradient background that changes over time
	r := uint8((frameNum * 2) % 255)
	g := uint8((frameNum * 3) % 255)
	b := uint8((frameNum * 5) % 255)

	bgColor := color.RGBA{r, g, b, 255}
	draw.Draw(img, img.Bounds(), &image.Uniform{bgColor}, image.Point{}, draw.Src)

	// Add a moving rectangle
	rectX := (frameNum * 5) % (WIDTH - 100)
	rectY := (frameNum * 3) % (HEIGHT - 100)
	rectBounds := image.Rect(rectX, rectY, rectX+100, rectY+100)
	rectColor := color.RGBA{255 - r, 255 - g, 255 - b, 255}
	draw.Draw(img, rectBounds, &image.Uniform{rectColor}, image.Point{}, draw.Src)

	// Encode as JPEG
	var buf []byte
	w := &byteWriter{buf: &buf}

	err := jpeg.Encode(w, img, &jpeg.Options{Quality: 80})
	if err != nil {
		log.Printf("Error encoding JPEG: %v", err)
		return nil
	}

	return buf
}

type byteWriter struct {
	buf *[]byte
}

func (w *byteWriter) Write(p []byte) (n int, err error) {
	*w.buf = append(*w.buf, p...)
	return len(p), nil
}

func streamMJPEG(w http.ResponseWriter, r *http.Request) {
	log.Println("Client connected for MJPEG stream")

	w.Header().Set("Content-Type", "multipart/x-mixed-replace; boundary=frame")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "close")

	fmt.Fprintf(w, "HTTP/1.1 200 OK\r\n")
	fmt.Fprintf(w, "Content-Type: multipart/x-mixed-replace; boundary=frame\r\n\r\n")

	flusher, ok := w.(http.Flusher)
	if !ok {
		log.Println("Streaming unsupported")
		return
	}

	frameNum := 0
	ticker := time.NewTicker(time.Second / FPS)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:

			frameData := generateDummyFrame(frameNum)
			if frameData == nil {
				continue
			}

			fmt.Fprintf(w, "--frame\r\n")
			fmt.Fprintf(w, "Content-Type: image/jpeg\r\n")
			fmt.Fprintf(w, "Content-Length: %d\r\n\r\n", len(frameData))

			_, err := w.Write(frameData)
			if err != nil {
				log.Printf("Error writing frame: %v", err)
				return
			}

			fmt.Fprintf(w, "\r\n")
			flusher.Flush()

			frameNum++

		case <-r.Context().Done():
			log.Println("Client disconnected")
			return
		}
	}
}

func main() {
	http.HandleFunc("/", streamMJPEG)

	log.Println("Dummy MJPEG server starting on http://localhost:8002")

	if err := http.ListenAndServe(":8002", nil); err != nil {
		log.Fatal("Server failed:", err)
	}
}
