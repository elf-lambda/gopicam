package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/zeebo/xxh3"
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

var serverState = &ServerState{
	serverStartTime:    int(time.Now().UnixMilli()),
	recordingStartTime: -1,
}

// -------------------------------------------------------------------------------------

var tmpl = template.Must(template.ParseFiles("templates/videos.html"))

type fileEntry struct {
	Name        string
	LinkEscaped string
}

type templateData struct {
	ParentLink        string
	ParentLinkEscaped string
	Dirs              []fileEntry
	Files             []fileEntry
}

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

		frameMutex.Lock()
		if !bytes.Equal(frameData, latestFrame) {
			latestFrame = frameData
		}
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
		// "-crf", "0",
		"-pix_fmt", "yuv420p",
		"-b:v", "1M",
		"-f", "segment",
		"-reset_timestamps", "1",
		"-segment_time", "1800",
		"-segment_format", "mkv",
		"-segment_atclocktime", "1",
		"-strftime", "1",
		config.recording_clips_dir + "/%Y%m%d/%Y%m%dT%H%M%S.mkv",
		// config.recording_clips_dir + "/%Y%m%dT%H%M%S.mkv",
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
		serverState.recordingStartTime = int(time.Now().UnixMilli())
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
		return
	}
	fmt.Println("Recording not started. Doing nothing.")
}

func formatSize(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := uint64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// -------------------------------------------------------------------------------------
// -------------------------------------------------------------------------------------

func streamHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "multipart/x-mixed-replace; boundary=frame")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	var lastHash uint64

	for {
		frameMutex.RLock()
		frame := make([]byte, len(latestFrame))
		copy(frame, latestFrame)
		frameMutex.RUnlock()

		if len(frame) == 0 {
			time.Sleep(10 * time.Millisecond)
			continue
		}

		currentHash := xxh3.Hash(frame)
		if currentHash == lastHash {
			time.Sleep(5 * time.Millisecond)
			continue
		}
		lastHash = currentHash

		_, err := fmt.Fprintf(w, "--frame\r\nContent-Type: image/jpeg\r\nContent-Length: %d\r\n\r\n", len(frame))
		if err != nil {
			log.Println("Client disconnected:", err)
			return
		}

		_, err = w.Write(frame)
		if err != nil {
			log.Println("Write error:", err)
			return
		}

		fmt.Fprint(w, "\r\n")
		flusher.Flush()
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
	if action == "stop" {
		fmt.Fprintf(w, "Not Recording..")
	} else {
		fmt.Fprintf(w, "Recording...")
	}
}

func statisticsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// TODO: change this on deployment
	diskStats := getDiskSpaceInfo(config.statisticsDir)

	diskInfo := map[string]interface{}{
		"totalSpace":               diskStats[0],
		"totalSpaceFormatted":      formatSize(diskStats[0]),
		"freeSpace":                diskStats[1],
		"freeSpaceFormatted":       formatSize(diskStats[1]),
		"usableSpace":              diskStats[2],
		"usableSpaceFormatted":     formatSize(diskStats[2]),
		"serverStartTimeMillis":    serverState.serverStartTime,
		"recordingStartTimeMillis": serverState.recordingStartTime,
	}

	w.Header().Set("Content-Type", "application/json")

	err := json.NewEncoder(w).Encode(diskInfo)
	if err != nil {
		http.Error(w, "Error encoding JSON", http.StatusInternalServerError)
		return
	}

	log.Println("Sent Statistics to client")
}

func deleteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Error parsing form", http.StatusBadRequest)
		return
	}

	days, err := strconv.ParseInt(r.FormValue("days"), 10, 64)
	if err != nil {
		http.Error(w, "Error parsing days", http.StatusBadRequest)
		return
	}
	deletedFiles := deleteFilesOlderThan(config.recording_clips_dir, int(days))

	w.Header().Set("Content-Type", "text")

	if deletedFiles == 0 {
		fmt.Fprintf(w, "0 Files Deleted")
	} else {
		fmt.Fprintf(w, "%d Files Deleted", deletedFiles)
	}
}

func videosHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	subPath := r.URL.Query().Get("path")
	fullPath := filepath.Join(config.recording_clips_dir, filepath.Clean(subPath))

	files, err := os.ReadDir(fullPath)
	if err != nil {
		http.Error(w, "Failed to read directory", http.StatusInternalServerError)
		return
	}

	var data templateData

	// add "Up" link
	if subPath != "" && subPath != "." {
		parent := filepath.Dir(subPath)
		data.ParentLink = parent
		data.ParentLinkEscaped = url.QueryEscape(parent)
	}

	// populate
	for _, entry := range files {
		if entry.IsDir() {
			dirName := entry.Name()
			linkPath := path.Join(subPath, dirName)
			data.Dirs = append(data.Dirs, fileEntry{
				Name:        dirName,
				LinkEscaped: url.QueryEscape(linkPath),
			})
		} else if entry.Type().IsRegular() && (strings.HasSuffix(strings.ToLower(entry.Name()), ".mkv") || strings.HasSuffix(strings.ToLower(entry.Name()), ".avi")) {
			fileName := entry.Name()
			linkPath := path.Join(subPath, fileName)
			data.Files = append(data.Files, fileEntry{
				Name:        fileName,
				LinkEscaped: url.PathEscape(linkPath),
			})
		}
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	err = tmpl.Execute(w, data)
	if err != nil {
		http.Error(w, "Template execution error", http.StatusInternalServerError)
	}
}
