import DefaultTheme from 'vitepress/theme'
import { h } from 'vue'
import './custom.css'

// Import custom components
import HeroVideo from './components/HeroVideo.vue'
import InteractiveDemo from './components/InteractiveDemo.vue'
import MetricsCard from './components/MetricsCard.vue'

export default {
  extends: DefaultTheme,
  Layout: () => {
    return h(DefaultTheme.Layout, null, {
      // Additional slots can be used here if needed
    })
  },
  enhanceApp({ app, router, siteData }) {
    // Register global components
    app.component('HeroVideo', HeroVideo)
    app.component('InteractiveDemo', InteractiveDemo)
    app.component('MetricsCard', MetricsCard)
    
    // Add smooth scroll behavior
    if (typeof window !== 'undefined') {
      router.onAfterRouteChanged = () => {
        // Scroll to top on route change
        window.scrollTo({ top: 0, behavior: 'smooth' })
      }
    }
  }
}