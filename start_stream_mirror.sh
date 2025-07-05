#!/bin/bash

# Unload the v4l2loopback module if it's already loaded
sudo modprobe -r v4l2loopback

# Load the v4l2loopback module with 2 devices, using video numbers 99 and 98 with labels
sudo modprobe v4l2loopback devices=2 video_nr=99,98 card_label="Virtual Video"

# Stream from /dev/video0 to virtual devices /dev/video99 and /dev/video98
nohup ffmpeg -f v4l2 -input_format mjpeg -video_size 1280x720 -i /dev/video0 -c:v copy -f v4l2 /dev/video98 -c:v copy -f v4l2 /dev/video99 &