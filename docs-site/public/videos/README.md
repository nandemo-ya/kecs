# KECS Demo Videos

This directory should contain the following video files for the documentation site:

## Required Videos

### kecs-demo.mp4 / kecs-demo.webm
- **Duration**: 30-60 seconds
- **Content**: Quick demonstration of KECS in action
- **Suggested scenes**:
  1. Starting KECS server
  2. Creating a cluster using AWS CLI
  3. Deploying a service
  4. Viewing the Web UI dashboard
  5. Real-time task updates

## Video Specifications

- **Resolution**: 1920x1080 (Full HD) or 1280x720 (HD)
- **Frame Rate**: 30fps
- **Formats**: 
  - MP4 (H.264) for broad compatibility
  - WebM (VP9) for modern browsers
- **File Size**: Keep under 10MB for fast loading
- **Optimization**: Use video compression tools to reduce file size

## Creating Demo Videos

You can create demo videos using:
- **Screen recording tools**: OBS Studio, QuickTime (Mac), or similar
- **Terminal recording**: asciinema for terminal sessions
- **Video editing**: FFmpeg for conversion and optimization

### Example FFmpeg commands:

```bash
# Convert to MP4
ffmpeg -i input.mov -c:v libx264 -preset slow -crf 22 -c:a aac -b:a 128k kecs-demo.mp4

# Convert to WebM
ffmpeg -i input.mov -c:v libvpx-vp9 -crf 30 -b:v 0 -b:a 128k -c:a libopus kecs-demo.webm

# Optimize for web (fast start)
ffmpeg -i kecs-demo.mp4 -movflags +faststart kecs-demo-optimized.mp4
```

## Placeholder

Until actual videos are created, the site will gracefully fall back to a gradient background with animations.