<template>
  <div class="metrics-grid">
    <div
      v-for="(metric, index) in metrics"
      :key="index"
      class="metric-card"
      :style="{ animationDelay: `${index * 0.1}s` }"
    >
      <div class="metric-icon">{{ metric.icon }}</div>
      <div class="metric-value" ref="valueRefs">{{ displayValues[index] }}{{ metric.suffix }}</div>
      <div class="metric-label">{{ metric.label }}</div>
      <div class="metric-description">{{ metric.description }}</div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, watch } from 'vue'
import { useData } from 'vitepress'

const props = defineProps({
  metrics: {
    type: Array,
    required: true,
    validator: (metrics) => {
      return metrics.every(m => 
        m.hasOwnProperty('icon') &&
        m.hasOwnProperty('value') &&
        m.hasOwnProperty('label')
      )
    }
  }
})

const { isDark } = useData()
const displayValues = ref(props.metrics.map(() => 0))
const valueRefs = ref([])

// Animate numbers counting up
const animateValue = (index, start, end, duration) => {
  const startTimestamp = performance.now()
  
  const step = (timestamp) => {
    const progress = Math.min((timestamp - startTimestamp) / duration, 1)
    const easeOutQuart = 1 - Math.pow(1 - progress, 4)
    const current = Math.floor(easeOutQuart * (end - start) + start)
    
    displayValues.value[index] = current
    
    if (progress < 1) {
      requestAnimationFrame(step)
    }
  }
  
  requestAnimationFrame(step)
}

// Intersection Observer for animation trigger
const observeMetrics = () => {
  const observer = new IntersectionObserver(
    (entries) => {
      entries.forEach((entry) => {
        if (entry.isIntersecting) {
          props.metrics.forEach((metric, index) => {
            animateValue(index, 0, metric.value, 2000)
          })
          observer.disconnect()
        }
      })
    },
    { threshold: 0.5 }
  )
  
  const metricsGrid = document.querySelector('.metrics-grid')
  if (metricsGrid) {
    observer.observe(metricsGrid)
  }
}

onMounted(() => {
  observeMetrics()
})

// Re-animate when theme changes
watch(isDark, () => {
  props.metrics.forEach((metric, index) => {
    animateValue(index, 0, metric.value, 1000)
  })
})
</script>

<style scoped>
.metrics-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
  gap: 1.5rem;
  margin: 3rem 0;
}

.metric-card {
  background: rgba(255, 255, 255, 0.03);
  backdrop-filter: blur(10px);
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 12px;
  padding: 2rem;
  text-align: center;
  position: relative;
  overflow: hidden;
  transition: all 0.3s ease;
  animation: slideUp 0.6s ease-out forwards;
  opacity: 0;
}

@keyframes slideUp {
  from {
    opacity: 0;
    transform: translateY(30px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.metric-card::before {
  content: "";
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  height: 3px;
  background: var(--kecs-gradient);
  transform: scaleX(0);
  transform-origin: left;
  transition: transform 0.5s ease;
}

.metric-card:hover {
  transform: translateY(-5px);
  border-color: var(--kecs-primary);
  box-shadow: 0 10px 30px rgba(102, 126, 234, 0.1);
}

.metric-card:hover::before {
  transform: scaleX(1);
}

.metric-icon {
  font-size: 3rem;
  margin-bottom: 1rem;
  filter: grayscale(0);
  transition: all 0.3s ease;
}

.metric-card:hover .metric-icon {
  transform: scale(1.1);
  filter: drop-shadow(0 0 10px rgba(102, 126, 234, 0.5));
}

.metric-value {
  font-size: 2.5rem;
  font-weight: 700;
  background: var(--kecs-gradient);
  -webkit-background-clip: text;
  background-clip: text;
  -webkit-text-fill-color: transparent;
  margin-bottom: 0.5rem;
  font-variant-numeric: tabular-nums;
}

.metric-label {
  font-size: 1.1rem;
  font-weight: 600;
  color: var(--vp-c-text-1);
  margin-bottom: 0.5rem;
}

.metric-description {
  font-size: 0.9rem;
  color: var(--vp-c-text-2);
  line-height: 1.5;
}

/* Dark mode adjustments */
.dark .metric-card {
  background: rgba(255, 255, 255, 0.02);
  border-color: rgba(255, 255, 255, 0.08);
}

.dark .metric-card:hover {
  box-shadow: 0 10px 30px rgba(0, 0, 0, 0.3);
}

/* Responsive design */
@media (max-width: 768px) {
  .metrics-grid {
    grid-template-columns: 1fr;
    gap: 1rem;
  }
  
  .metric-card {
    padding: 1.5rem;
  }
  
  .metric-value {
    font-size: 2rem;
  }
}

/* Accessibility */
@media (prefers-reduced-motion: reduce) {
  .metric-card {
    animation: none;
    opacity: 1;
  }
  
  .metric-value {
    transition: none;
  }
}
</style>