package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os/exec"
	"sync"
	"time"
)

var (
	latestFrame []byte
	frameMutex  sync.RWMutex
	config      = ReadConfig("gopicam.conf")
)

// -------------------------------------------------------------------------------------

type ServerState struct {
	recordingState     bool
	serverStartTime    int
	recordingStartTime int
	ffmpegPid          int
	ffmpegCmd          *exec.Cmd
}

var serverState = &ServerState{serverStartTime: int(time.Now().Unix())}

// -------------------------------------------------------------------------------------

func readFrames(reader io.Reader) {
	mr := multipart.NewReader(reader, "frame")

	for {
		part, err := mr.NextPart()
		if err != nil {
			return
		}

		frameData, err := io.ReadAll(part)
		if err != nil {
			continue
		}

		// Store the frame
		frameMutex.Lock()
		latestFrame = frameData
		frameMutex.Unlock()
	}
}

func connectToSource(source string) {
	for {
		resp, err := http.Get(source)
		if err != nil {
			log.Printf("Error connecting to source: %v", err)
			continue
		}

		log.Println("Connected to MJPEG source ", source)
		readFrames(resp.Body)
		resp.Body.Close()

		log.Println("Connection lost, reconnecting...")
	}
}

func getFFMPEGCommand(config *Config) []string {
	command := []string{
		"ffmpeg",
		"-nostdin",
		"-f", "v4l2",
		"-framerate", "30",
		"-video_size", "1280x720",
		"-i", config.camera_url,
		"-c:v", "h264_v4l2m2m",
		"-crf", "0",
		"-pix_fmt", "yuv420p",
		"-b:v", "1M",
		"-f", "segment",
		"-reset_timestamps", "1",
		"-segment_time", "1800",
		"-segment_format", "mkv",
		"-segment_atclocktime", "1",
		"-strftime", "1",
		config.recording_clips_dir + "/%Y%m%dT%H%M%S.mkv",
	}

	return command
}

func startFFMPEGRecording() error {
	if !serverState.recordingState {
		serverState.ffmpegCmd = exec.Command(getFFMPEGCommand(config)[0], getFFMPEGCommand(config)[1:]...)
		err := serverState.ffmpegCmd.Start()
		if err != nil {
			log.Fatal(err)
		}
		serverState.recordingStartTime = int(time.Now().Unix())
		serverState.ffmpegPid = serverState.ffmpegCmd.Process.Pid
		serverState.recordingState = true

		fmt.Printf("FFmpeg started with PID: %d\n", serverState.ffmpegCmd.Process.Pid)
		return nil
	}
	return errors.New("recording already started")
}

func stopFFMPEGRecording() {
	if serverState.recordingState {
		fmt.Printf("Stopping ffmpeg recording with PID: %d\n", serverState.ffmpegPid)
		if serverState.ffmpegCmd != nil && serverState.ffmpegCmd.Process != nil {
			serverState.ffmpegCmd.Process.Kill()
		}
		serverState.recordingState = false
		serverState.recordingStartTime = -1
		serverState.ffmpegCmd = nil
		serverState.ffmpegPid = -1
	}
	fmt.Println("Recording not started. Doing nothing.")
}

// -------------------------------------------------------------------------------------
// -------------------------------------------------------------------------------------

func streamHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "multipart/x-mixed-replace; boundary=frame")

	flusher := w.(http.Flusher)

	for {
		frameMutex.RLock()
		frame := make([]byte, len(latestFrame))
		copy(frame, latestFrame)
		frameMutex.RUnlock()

		if len(frame) == 0 {
			continue
		}

		fmt.Fprintf(w, "--frame\r\nContent-Type: image/jpeg\r\nContent-Length: %d\r\n\r\n", len(frame))
		w.Write(frame)
		fmt.Fprintf(w, "\r\n")
		flusher.Flush()

		time.Sleep(33 * time.Millisecond) // 30fps
	}
}

func recordHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Error parsing form", http.StatusBadRequest)
		return
	}

	action := r.FormValue("action")

	switch action {
	case "start":
		startFFMPEGRecording()
		fmt.Println("Starting recording...")

	case "stop":
		stopFFMPEGRecording()
		fmt.Println("Stopping recording...")

	default:
		http.Error(w, "Invalid action", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"status": "success", "action": "%s"}`, action)
}

func statisticsHandler(w http.ResponseWriter, r *http.Request) {

}
