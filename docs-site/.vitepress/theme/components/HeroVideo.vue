<template>
  <div class="hero-video-wrapper" v-if="showVideo">
    <div class="hero-video-container">
      <video
        ref="videoElement"
        class="hero-video"
        autoplay
        muted
        loop
        playsinline
        @loadeddata="onVideoLoaded"
        @error="onVideoError"
      >
        <source :src="withBase('/videos/kecs-demo.mp4')" type="video/mp4">
        <source :src="withBase('/videos/kecs-demo.webm')" type="video/webm">
        Your browser does not support the video tag.
      </video>
      <div class="video-overlay"></div>
    </div>
    <div class="hero-content-overlay">
      <slot></slot>
    </div>
  </div>
  <div v-else class="hero-fallback">
    <div class="hero-fallback-background"></div>
    <div class="hero-content-overlay">
      <slot></slot>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import { withBase } from 'vitepress'

const videoElement = ref(null)
const showVideo = ref(false)
const videoLoaded = ref(false)

// Check if user prefers reduced motion
const prefersReducedMotion = () => {
  return window.matchMedia('(prefers-reduced-motion: reduce)').matches
}

// Check if device can handle video playback well
const canPlayVideo = () => {
  // Don't play video on mobile devices to save bandwidth
  const isMobile = /Android|webOS|iPhone|iPad|iPod|BlackBerry|IEMobile|Opera Mini/i.test(navigator.userAgent)
  
  // Don't play video if user prefers reduced motion
  if (prefersReducedMotion()) return false
  
  // Don't play video on mobile
  if (isMobile) return false
  
  // Check connection speed if available
  if ('connection' in navigator) {
    const connection = navigator.connection
    if (connection.saveData) return false
    if (connection.effectiveType && ['slow-2g', '2g'].includes(connection.effectiveType)) return false
  }
  
  return true
}

const onVideoLoaded = () => {
  videoLoaded.value = true
}

const onVideoError = (error) => {
  console.warn('Video failed to load:', error)
  showVideo.value = false
}

onMounted(() => {
  showVideo.value = canPlayVideo()
  
  // Listen for preference changes
  const mediaQuery = window.matchMedia('(prefers-reduced-motion: reduce)')
  const handleChange = (e) => {
    showVideo.value = !e.matches && canPlayVideo()
  }
  
  mediaQuery.addEventListener('change', handleChange)
  
  onUnmounted(() => {
    mediaQuery.removeEventListener('change', handleChange)
  })
})
</script>

<style scoped>
.hero-video-wrapper,
.hero-fallback {
  position: relative;
  width: 100%;
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
}

.hero-video-container {
  position: absolute;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  z-index: -2;
}

.hero-video {
  position: absolute;
  top: 50%;
  left: 50%;
  min-width: 100%;
  min-height: 100%;
  width: auto;
  height: auto;
  transform: translateX(-50%) translateY(-50%);
  object-fit: cover;
}

.video-overlay {
  position: absolute;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  background: linear-gradient(
    135deg,
    rgba(102, 126, 234, 0.7) 0%,
    rgba(118, 75, 162, 0.7) 100%
  );
  z-index: 1;
}

.hero-fallback-background {
  position: absolute;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  background: var(--kecs-gradient);
  opacity: 0.9;
  z-index: -1;
}

.hero-fallback-background::before {
  content: "";
  position: absolute;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  background-image: 
    repeating-linear-gradient(
      45deg,
      transparent,
      transparent 35px,
      rgba(255, 255, 255, 0.05) 35px,
      rgba(255, 255, 255, 0.05) 70px
    );
  animation: stripe-move 20s linear infinite;
}

@keyframes stripe-move {
  0% { transform: translateX(0); }
  100% { transform: translateX(70px); }
}

.hero-content-overlay {
  position: relative;
  z-index: 2;
  width: 100%;
  max-width: 1200px;
  padding: 0 24px;
  text-align: center;
}

/* Ensure content is visible */
:deep(.VPHero) {
  background: transparent !important;
}

/* Add loading state */
.hero-video-container::before {
  content: "";
  position: absolute;
  top: 50%;
  left: 50%;
  width: 40px;
  height: 40px;
  margin: -20px 0 0 -20px;
  border: 3px solid rgba(255, 255, 255, 0.3);
  border-top-color: white;
  border-radius: 50%;
  animation: spin 1s linear infinite;
  z-index: 2;
  opacity: 0;
  transition: opacity 0.3s;
}

.hero-video-container:not(.loaded)::before {
  opacity: 1;
}

@keyframes spin {
  0% { transform: rotate(0deg); }
  100% { transform: rotate(360deg); }
}

/* Responsive adjustments */
@media (max-width: 768px) {
  .hero-video-wrapper,
  .hero-fallback {
    min-height: 80vh;
  }
}
</style>