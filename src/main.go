package main

import (
	"fmt"
	"log"
	"net/http"
)

// -------------------------------------------------------------------------------------

func main() {
	go connectToSource("http://localhost:8002")
	fmt.Println("Loaded config: ", config)

	fs := http.FileServer(http.Dir("static"))
	http.Handle("/", fs)
	http.HandleFunc("/stream", streamHandler)
	http.HandleFunc("/record", recordHandler)
	http.HandleFunc("/statistics", statisticsHandler)
	http.HandleFunc("/delete", deleteHandler)

	log.Println("Server starting on http://localhost:8001")
	log.Fatal(http.ListenAndServe(":8001", nil))
}
