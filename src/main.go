package main

import (
	"fmt"
	"log"
	"net/http"

	_ "net/http/pprof"
)

// -------------------------------------------------------------------------------------

func main() {
	// DEBUG
	// go func() {
	// 	log.Println(http.ListenAndServe("localhost:6060", nil))
	// }()

	// Port 8080 for live
	go connectToSource("http://localhost:8080")
	go scheduleFFmpegRollover()
	fmt.Println("Loaded config: ", config)

	fs := http.FileServer(http.Dir("static"))
	http.Handle("/", fs)
	http.HandleFunc("/stream", streamHandler)
	http.HandleFunc("/record", recordHandler)
	http.HandleFunc("/statistics", statisticsHandler)
	http.HandleFunc("/delete", deleteHandler)
	http.HandleFunc("/videos", videosHandler)
	http.Handle("/clips/", http.StripPrefix("/clips/", http.FileServer(http.Dir(config.recording_clips_dir))))

	log.Println("Server starting on http://localhost:8001")
	log.Fatal(http.ListenAndServe(":8001", nil))
}
