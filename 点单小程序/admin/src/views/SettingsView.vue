<template>
  <div>
    <div class="page-heading">
      <div><h1>店铺设置</h1><p>管理营业信息和快速小程序封面。</p></div>
      <ShopQR/>
    </div>
    <section v-loading="loading" class="atelier-panel settings-panel">
      <div class="panel-toolbar"><h2>店铺信息</h2><el-button type="text" :disabled="saving" @click="load">重新加载</el-button></div>
      <el-form label-position="top" @submit.native.prevent="save">
        <el-form-item label="店铺名称"><el-input v-model.trim="form.name" maxlength="30"/></el-form-item>
        <el-form-item label="店铺公告"><el-input v-model.trim="form.announcement" type="textarea" :rows="2" maxlength="240"/></el-form-item>
        <el-form-item label="快速小程序封面">
          <div class="settings-cover-preview"><img :src="preview || form.quickCover || '/images/hero.jpg'" alt="快速小程序封面预览"></div>
          <input ref="cover" type="file" accept="image/jpeg,image/png,image/webp,image/gif" hidden @change="selectCover">
          <el-button size="small" :disabled="saving" @click="$refs.cover.click()">上传封面</el-button>
          <el-button v-if="preview || form.quickCover" size="small" :disabled="saving" @click="clearCover">使用默认封面</el-button>
          <p class="form-hint">支持 JPG、PNG、WebP、GIF，最多 4 MB。建议上传横向照片。</p>
        </el-form-item>
        <el-form-item label="自提地点"><el-input v-model.trim="form.location" maxlength="80"/></el-form-item>
        <el-form-item label="营业时间"><el-input v-model.trim="form.hours" maxlength="40"/></el-form-item>
        <el-form-item label="接受新预约"><el-switch v-model="form.open" active-text="接受预约" inactive-text="暂停预约"/></el-form-item>
        <el-button type="primary" native-type="submit" :loading="saving">保存店铺设置</el-button>
      </el-form>
    </section>
  </div>
</template>

<script>
import ShopQR from '@/components/ShopQR.vue'
import { loadCatalog, saveSettings, uploadImage } from '@/api/catalog'

export default {
  components: { ShopQR },
  data: () => ({
    form: { name: '', announcement: '', location: '', hours: '', quickCover: '', open: false, version: 0 },
    loading: true, saving: false, selectedCover: null, preview: '',
  }),
  mounted() { this.load() },
  beforeDestroy() { this.clearSelection() },
  methods: {
    async load() {
      this.loading = true
      try {
        this.form = { quickCover: '', ...(await loadCatalog()).settings }
        this.clearSelection()
      } catch (error) { this.$message.error(error.message) }
      finally { this.loading = false }
    },
    selectCover(event) {
      const file = event.target.files[0]
      if (!file) return
      this.clearSelection()
      if (file.size > 4 * 1024 * 1024 || (file.type && !['image/jpeg', 'image/png', 'image/webp', 'image/gif'].includes(file.type))) {
        this.$message.error('请选择 4 MB 以内的 JPG、PNG、WebP 或 GIF 图片')
        return
      }
      this.selectedCover = file
      this.preview = URL.createObjectURL(file)
    },
    clearSelection() {
      if (this.preview) URL.revokeObjectURL(this.preview)
      this.preview = ''
      this.selectedCover = null
      if (this.$refs.cover) this.$refs.cover.value = ''
    },
    clearCover() {
      this.clearSelection()
      this.form.quickCover = ''
    },
    async save() {
      if (this.saving || this.loading) return
      if (!this.form.name) { this.$message.warning('请填写店铺名称'); return }
      this.saving = true
      try {
        if (this.selectedCover) {
          this.form.quickCover = (await uploadImage(this.selectedCover)).url
          this.clearSelection()
        }
        this.form = await saveSettings(this.form)
        this.$message.success('店铺设置已保存')
      } catch (error) { this.$message.error(error.message) }
      finally { this.saving = false }
    },
  },
}
</script>
