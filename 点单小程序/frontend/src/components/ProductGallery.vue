<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import Icon from '@/components/Icon.vue'

const props = defineProps<{ images: string[]; name: string }>()
const index = ref(0)
const zoomed = ref(false)
const current = computed(() => props.images[index.value] || props.images[0])
let start: { x: number; y: number } | undefined

watch(() => props.images.length, length => {
  if (index.value >= length) choose(0)
})

function choose(next: number) {
  index.value = next
  zoomed.value = false
}
function step(offset: number) {
  const count = props.images.length
  if (count > 1) choose((index.value + offset + count) % count)
}
function touchStart(event: TouchEvent) {
  const touch = event.touches[0]
  start = touch && event.touches.length === 1 && !zoomed.value
    ? { x: touch.clientX, y: touch.clientY } : undefined
}
function touchEnd(event: TouchEvent) {
  const touch = event.changedTouches[0]
  if (start && touch) {
    const dx = touch.clientX - start.x
    if (Math.abs(dx) >= 48 && Math.abs(dx) > 1.4 * Math.abs(touch.clientY - start.y)) step(dx < 0 ? 1 : -1)
  }
  start = undefined
}
</script>

<template>
  <section class="detail-gallery" aria-label="商品图片">
    <div class="detail-image" :class="{ zoomed }"
      @touchstart.passive="touchStart" @touchend.passive="touchEnd" @touchcancel="start = undefined">
      <img v-if="current" :src="current" :alt="name + (images.length > 1 ? ' · 图片 ' + (index + 1) : '')" draggable="false">
      <template v-if="images.length > 1">
        <button type="button" class="gallery-arrow gallery-prev" aria-label="上一张商品图" @click="step(-1)"><Icon name="chevron" :size="16" /></button>
        <button type="button" class="gallery-arrow gallery-next" aria-label="下一张商品图" @click="step(1)"><Icon name="chevron" :size="16" /></button>
        <span class="gallery-position" aria-live="polite">{{ index + 1 }} / {{ images.length }}</span>
      </template>
      <button v-if="current" type="button" class="icon-button detail-zoom" :aria-label="zoomed ? '缩小商品图' : '放大商品图'" :aria-pressed="zoomed" @click="zoomed = !zoomed"><Icon name="expand" :size="18" /></button>
    </div>
    <div v-if="images.length > 1" class="gallery-thumbnails" aria-label="选择商品图片">
      <button v-for="(image, number) in images" :key="image" type="button"
        :aria-label="'查看商品图 ' + (number + 1)" :aria-pressed="index === number" @click="choose(number)">
        <img :src="image" alt="" loading="lazy">
      </button>
    </div>
  </section>
</template>
