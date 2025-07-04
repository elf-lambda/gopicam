
#include <arpa/inet.h>
#include <fcntl.h>
#include <linux/videodev2.h>
#include <signal.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <sys/ioctl.h>
#include <sys/socket.h>
#include <unistd.h>

#define PORT 8080
#define DEVICE "/dev/video99"
#define FRAME_WIDTH 1280
#define FRAME_HEIGHT 720

/*
    The /dev/video99 is assumed to be a ffmpeg mirrored v4l2 api compliant
   stream from /dev/video0

    The architecture currently is using ffmpeg to mirror /dev/video0 to both
    /dev/video98 and 99, 98 is used for recording and 99 for streaming.

    This way we setup a simple C server to capture the frames from /dev/video99
    and stream them to a (1) single client, the Java server, that then handles
    multiple clients at the same time.

*/

int setup_device(const char *device, int *frame_size) {
  int fd = open(device, O_RDWR);
  if (fd < 0) {
    perror("open");
    return -1;
  }

  struct v4l2_format fmt = {0};
  fmt.type = V4L2_BUF_TYPE_VIDEO_CAPTURE;
  fmt.fmt.pix.width = FRAME_WIDTH;
  fmt.fmt.pix.height = FRAME_HEIGHT;
  fmt.fmt.pix.pixelformat = V4L2_PIX_FMT_MJPEG;
  fmt.fmt.pix.field = V4L2_FIELD_NONE;

  if (ioctl(fd, VIDIOC_S_FMT, &fmt) < 0) {
    perror("VIDIOC_S_FMT");
    close(fd);
    return -1;
  }

  *frame_size = fmt.fmt.pix.sizeimage;
  return fd;
}

int setup_server() {
  int server_fd = socket(AF_INET, SOCK_STREAM, 0);
  if (server_fd < 0) {
    perror("socket");
    return -1;
  }

  struct sockaddr_in addr = {0};
  addr.sin_family = AF_INET;
  addr.sin_addr.s_addr = INADDR_ANY;
  addr.sin_port = htons(PORT);

  if (bind(server_fd, (struct sockaddr *)&addr, sizeof(addr)) < 0) {
    perror("bind");
    close(server_fd);
    return -1;
  }

  if (listen(server_fd, 1) < 0) {
    perror("listen");
    close(server_fd);
    return -1;
  }

  return server_fd;
}

void stream_mjpeg(int client_fd, int dev_fd, int frame_size) {
  char http_hdr[] =
      "HTTP/1.1 200 OK\r\n"
      "Content-Type: multipart/x-mixed-replace; boundary=frame\r\n\r\n";
  send(client_fd, http_hdr, strlen(http_hdr), MSG_NOSIGNAL);

  unsigned char *frame = malloc(frame_size);
  char header[128];

  while (1) {
    int n = read(dev_fd, frame, frame_size);
    if (n <= 0)
      break;

    int header_len = snprintf(header, sizeof(header),
                              "--frame\r\n"
                              "Content-Type: image/jpeg\r\n"
                              "Content-Length: %d\r\n\r\n",
                              n);

    if (send(client_fd, header, header_len, MSG_NOSIGNAL) <= 0)
      break;
    if (send(client_fd, frame, n, MSG_NOSIGNAL) <= 0)
      break;
    if (send(client_fd, "\r\n", 2, MSG_NOSIGNAL) <= 0)
      break;

    usleep(33333); // ~30 FPS
  }

  free(frame);
  close(client_fd);
  printf("Client disconnected\n");
}

int main() {
  signal(SIGPIPE, SIG_IGN);

  int frame_size;
  int dev_fd = setup_device(DEVICE, &frame_size);
  if (dev_fd < 0)
    return 1;

  int server_fd = setup_server();
  if (server_fd < 0) {
    close(dev_fd);
    return 1;
  }

  printf("Streaming on http://localhost:%d\n", PORT);

  while (1) {
    int client_fd = accept(server_fd, NULL, NULL);
    if (client_fd >= 0) {
      printf("Client connected\n");
      stream_mjpeg(client_fd, dev_fd, frame_size);
    }
  }

  close(dev_fd);
  close(server_fd);
  return 0;
}
