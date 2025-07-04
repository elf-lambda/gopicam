package main

import (
	"bufio"
	"log"
	"os"
	"strings"
)

type Config struct {
	camera_url          string
	ffmpeg_log_file     string
	recording_clips_dir string
}

func ReadConfig(config_path string) *Config {
	file, err := os.Open(config_path)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	var tmp []string = make([]string, 0, 3)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		tokens := strings.Split(line, ":")
		key := tokens[0]
		value := tokens[1]
		switch key {
		case "camera_url":
			tmp = append(tmp, value)
		case "ffmpeg_log_file":
			tmp = append(tmp, value)
		case "recording_clips_dir":
			tmp = append(tmp, value)
		}
	}

	if len(tmp) < 3 {
		return &Config{
			camera_url:          "/dev/video98",
			ffmpeg_log_file:     "./ffmpeg.log",
			recording_clips_dir: "./clips",
		}
	}
	return &Config{tmp[0], tmp[1], tmp[2]}
}
