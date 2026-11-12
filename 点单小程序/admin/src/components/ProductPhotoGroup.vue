<template>
  <div class="photo-group">
    <ul v-if="photos.length" class="photo-list">
      <li v-for="(photo, index) in photos" :key="photo.url" class="photo-tile">
        <div class="photo-thumbnail">
          <img :src="photo.url" :alt="label + '第 ' + (index + 1) + ' 张图'">
          <span v-if="index === 0" class="photo-badge">{{ primaryLabel }}</span>
        </div>
        <div class="photo-tile-actions">
          <button v-if="index > 0" type="button" :disabled="disabled" @click="$emit('primary', index)">设为首图</button>
          <button type="button" :disabled="disabled" :aria-label="'移除' + label + '第 ' + (index + 1) + ' 张图'" @click="$emit('remove', index)">移除</button>
        </div>
      </li>
    </ul>
    <div class="photo-toolbar">
      <input ref="files" type="file" hidden multiple accept="image/jpeg,image/png,image/webp,image/gif" @change="choose">
      <el-button size="small" icon="el-icon-plus" :disabled="disabled || photos.length >= 8" @click="$refs.files.click()">添加图片</el-button>
      <span class="photo-count">{{ photos.length }} / 8</span>
      <button v-if="photos.length < 8" type="button" class="photo-link" :disabled="disabled" :aria-expanded="showLink" @click="showLink = !showLink">添加图片链接</button>
    </div>
    <div v-if="showLink && photos.length < 8" class="photo-link-fields">
      <el-input v-model.trim="link" :disabled="disabled" maxlength="1000" placeholder="HTTPS 图片地址"
        :aria-label="label + '图片地址'" @keydown.enter.native.prevent="addLink" />
      <el-button size="small" :disabled="disabled || !link" @click="addLink">添加</el-button>
    </div>
  </div>
</template>

<script>
export default {
  props: {
    photos: { type: Array, required: true },
    label: { type: String, required: true },
    primaryLabel: { type: String, default: '首图' },
    disabled: Boolean,
  },
  data: () => ({ link: '', showLink: false }),
  methods: {
    choose(event) {
      const files = Array.from(event.target.files)
      event.target.value = ''
      if (files.length && !this.disabled) this.$emit('files', files)
    },
    addLink() {
      if (!this.link || this.disabled) return
      this.$emit('link', this.link)
      this.link = ''
    },
  },
}
</script>
