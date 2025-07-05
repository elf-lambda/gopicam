# gopicam - Relay & Recorder

A lightweight webcam relay and recording system built in Go, designed primarily for Raspberry Pi environments. This project captures and streams a webcam feed efficiently, with multi-client support and segment-based video recording.

---

## Overview

-   Reads from a V4L2 video device (e.g., `/dev/video0` or a mirrored device via `v4l2loopback`).
-   Serves an MJPEG stream directly over HTTP for multi-client access.
-   Provides a simple web interface to view live video streams and browse recorded clips.
-   Manages video recording via FFmpeg with automated segmenting into 30-minute MKV files.
-   Supports on-demand start/stop of recordings through HTTP endpoints.
-   Implements folder-based retention cleanup to automatically manage disk space.

---

## Architecture

1. **Video Capture**  
   C Server that reads raw or mirrored video from a Linux V4L2 device.

2. **Go HTTP Server**

    - Serves live MJPEG streams efficiently to multiple clients on `/stream`.
    - Hosts a basic UI homepage and video listing page.
    - Exposes RESTful endpoints for starting/stopping recording sessions.
    - Streams recorded video clips for download.
    - Serves static assets (CSS, JS) from a dedicated folder.

3. **FFmpeg Integration**
    - Launches and manages FFmpeg as an external process.
    - Segments recordings into 30-minute MKV files, stored in date-organized folders.
    - Uses hardware acceleration where available (`h264_v4l2m2m` codec).

---

## Features

-   Multi-client MJPEG relay server with minimal resource usage.
-   Date-organized video clip storage with retention policy support.
-   Simple, responsive web UI with live preview and clip management.
-   Clean and maintainable Go codebase, leveraging native libraries and idiomatic concurrency.
-   Designed for headless Raspberry Pi deployments with minimal dependencies.

---

## Usage

1. **Run the start_stream_mirror.sh**

2. **Compile and run the C server**

```bash
gcc src/c/mjpeg_streaming_server.c -o mjpegserver
nohup ./mjpegserver &
```

3. **Run the Go server**

```bash
go build ./src/.
```

```bash
./gopicam
OR
nohup ./gopicam &
```
