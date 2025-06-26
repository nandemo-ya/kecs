<template>
  <div class="interactive-demo">
    <h3>{{ title }}</h3>
    <div class="demo-tabs">
      <button
        v-for="(tab, index) in tabs"
        :key="index"
        :class="['tab-button', { active: activeTab === index }]"
        @click="activeTab = index"
      >
        {{ tab.label }}
      </button>
    </div>
    <div class="demo-content">
      <Transition name="fade" mode="out-in">
        <div :key="activeTab" class="tab-content">
          <div class="code-wrapper">
            <div class="code-header">
              <span class="language-label">{{ tabs[activeTab].language }}</span>
              <button class="copy-button" @click="copyCode" :class="{ copied }">
                <span v-if="!copied">📋 Copy</span>
                <span v-else>✅ Copied!</span>
              </button>
            </div>
            <pre class="code-block"><code>{{ tabs[activeTab].code }}</code></pre>
          </div>
          <div v-if="tabs[activeTab].output" class="output-section">
            <div class="output-header">Output</div>
            <pre class="output-block"><code>{{ tabs[activeTab].output }}</code></pre>
          </div>
        </div>
      </Transition>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'

const props = defineProps({
  title: {
    type: String,
    default: 'Interactive Demo'
  },
  tabs: {
    type: Array,
    required: true,
    validator: (tabs) => {
      return tabs.every(tab => 
        tab.hasOwnProperty('label') && 
        tab.hasOwnProperty('code') &&
        tab.hasOwnProperty('language')
      )
    }
  }
})

const activeTab = ref(0)
const copied = ref(false)

const copyCode = async () => {
  const code = props.tabs[activeTab.value].code
  try {
    await navigator.clipboard.writeText(code)
    copied.value = true
    setTimeout(() => {
      copied.value = false
    }, 2000)
  } catch (err) {
    console.error('Failed to copy:', err)
  }
}
</script>

<style scoped>
.interactive-demo {
  background: rgba(255, 255, 255, 0.02);
  backdrop-filter: blur(10px);
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 12px;
  padding: 0;
  margin: 2rem 0;
  overflow: hidden;
  transition: all 0.3s ease;
}

.interactive-demo:hover {
  border-color: var(--kecs-primary);
  box-shadow: 0 0 30px rgba(102, 126, 234, 0.1);
}

.interactive-demo h3 {
  margin: 0;
  padding: 1.5rem 2rem;
  background: rgba(255, 255, 255, 0.03);
  border-bottom: 1px solid rgba(255, 255, 255, 0.05);
  font-size: 1.2rem;
}

.demo-tabs {
  display: flex;
  background: rgba(255, 255, 255, 0.02);
  border-bottom: 1px solid rgba(255, 255, 255, 0.05);
  overflow-x: auto;
}

.tab-button {
  padding: 1rem 2rem;
  background: transparent;
  border: none;
  color: var(--vp-c-text-2);
  cursor: pointer;
  font-size: 0.9rem;
  font-weight: 500;
  transition: all 0.3s ease;
  position: relative;
  white-space: nowrap;
}

.tab-button:hover {
  color: var(--vp-c-text-1);
  background: rgba(255, 255, 255, 0.05);
}

.tab-button.active {
  color: var(--kecs-primary);
  background: rgba(102, 126, 234, 0.1);
}

.tab-button.active::after {
  content: "";
  position: absolute;
  bottom: 0;
  left: 0;
  right: 0;
  height: 2px;
  background: var(--kecs-gradient);
}

.demo-content {
  padding: 2rem;
}

.tab-content {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}

.code-wrapper {
  position: relative;
  border-radius: 8px;
  overflow: hidden;
  background: var(--vp-code-block-bg);
  border: 1px solid rgba(255, 255, 255, 0.05);
}

.code-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0.75rem 1rem;
  background: rgba(255, 255, 255, 0.03);
  border-bottom: 1px solid rgba(255, 255, 255, 0.05);
}

.language-label {
  font-size: 0.8rem;
  color: var(--kecs-primary);
  font-weight: 600;
  text-transform: uppercase;
}

.copy-button {
  padding: 0.25rem 0.75rem;
  background: rgba(255, 255, 255, 0.1);
  border: 1px solid rgba(255, 255, 255, 0.2);
  border-radius: 4px;
  color: var(--vp-c-text-2);
  font-size: 0.8rem;
  cursor: pointer;
  transition: all 0.2s ease;
}

.copy-button:hover {
  background: rgba(255, 255, 255, 0.2);
  color: var(--vp-c-text-1);
}

.copy-button.copied {
  background: rgba(72, 187, 120, 0.2);
  border-color: rgba(72, 187, 120, 0.5);
  color: #48bb78;
}

.code-block,
.output-block {
  margin: 0;
  padding: 1.5rem;
  overflow-x: auto;
}

.code-block code,
.output-block code {
  font-family: var(--vp-font-family-mono);
  font-size: 0.875rem;
  line-height: 1.6;
}

.output-section {
  position: relative;
  border-radius: 8px;
  overflow: hidden;
  background: rgba(72, 187, 120, 0.05);
  border: 1px solid rgba(72, 187, 120, 0.2);
}

.output-header {
  padding: 0.75rem 1rem;
  background: rgba(72, 187, 120, 0.1);
  border-bottom: 1px solid rgba(72, 187, 120, 0.2);
  font-size: 0.8rem;
  font-weight: 600;
  color: #48bb78;
  text-transform: uppercase;
}

/* Transitions */
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.3s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}

/* Responsive */
@media (max-width: 768px) {
  .demo-tabs {
    -webkit-overflow-scrolling: touch;
  }
  
  .tab-button {
    padding: 0.75rem 1.5rem;
    font-size: 0.85rem;
  }
  
  .demo-content {
    padding: 1.5rem;
  }
  
  .code-block,
  .output-block {
    padding: 1rem;
  }
}

/* Scrollbar styling */
.demo-tabs::-webkit-scrollbar,
.code-block::-webkit-scrollbar,
.output-block::-webkit-scrollbar {
  height: 6px;
}

.demo-tabs::-webkit-scrollbar-track,
.code-block::-webkit-scrollbar-track,
.output-block::-webkit-scrollbar-track {
  background: rgba(255, 255, 255, 0.05);
}

.demo-tabs::-webkit-scrollbar-thumb,
.code-block::-webkit-scrollbar-thumb,
.output-block::-webkit-scrollbar-thumb {
  background: rgba(255, 255, 255, 0.2);
  border-radius: 3px;
}

.demo-tabs::-webkit-scrollbar-thumb:hover,
.code-block::-webkit-scrollbar-thumb:hover,
.output-block::-webkit-scrollbar-thumb:hover {
  background: rgba(255, 255, 255, 0.3);
}
</style>