<template>
  <div>
    <div class="page-heading">
      <div><h1>商品与库存</h1><p>创建分类后，在对应分类中添加商品。</p></div>
      <el-button type="primary" icon="el-icon-plus" :disabled="!categories.length || loading" @click="edit()">添加商品</el-button>
    </div>

    <section v-loading="loading" class="atelier-panel catalog-panel">
      <div class="panel-toolbar">
        <h2>商品分类</h2>
        <el-button icon="el-icon-plus" @click="editCategory()">新增分类</el-button>
      </div>
      <div v-if="categories.length" class="category-manager">
        <button class="category-filter" :aria-pressed="categoryId === null" @click="categoryId = null">全部商品（{{ products.length }}）</button>
        <div v-for="category in categories" :key="category.id" class="category-row">
          <button class="category-filter" :aria-pressed="categoryId === category.id" @click="categoryId = category.id">
            {{ category.name }}（{{ categoryCounts[category.id] || 0 }}）
          </button>
          <el-button type="text" :disabled="categoryWorking" @click="editCategory(category)">改名</el-button>
          <el-button type="text" class="danger-text" :disabled="categoryWorking" @click="removeCategory(category)">删除</el-button>
        </div>
      </div>
      <div v-else class="panel-empty">暂无分类。新增分类后即可添加商品。</div>
    </section>

    <section class="atelier-panel">
      <div class="panel-toolbar">
        <h2>{{ activeCategory ? activeCategory.name : '商品目录' }} <small>{{ filtered.length }} 件</small></h2>
        <el-input v-model="search" prefix-icon="el-icon-search" placeholder="搜索商品名称或分类" clearable />
      </div>
      <div v-loading="loading">
        <ul v-if="filtered.length" class="inventory-list" aria-label="商品目录">
          <li v-for="product in filtered" :key="product.id" class="inventory-row">
            <div class="table-artwork">
              <img :src="product.image" :alt="product.name" loading="lazy">
              <div><strong>{{ product.name }}</strong><span>{{ product.category }}</span></div>
            </div>
            <span class="inventory-price">¥ {{ money(product.price) }}</span>
            <span class="inventory-stock">库存 {{ product.stock }}</span>
            <el-tag size="small" :type="product.available ? 'success' : 'info'" effect="plain">
              {{ product.available ? '在售' : '已下架' }}
            </el-tag>
            <div class="inventory-actions">
              <el-button type="text" @click="edit(product)">编辑</el-button>
              <el-button type="text" :disabled="!!workingProduct" @click="toggle(product)">{{ product.available ? '下架' : '上架' }}</el-button>
            </div>
          </li>
        </ul>
        <div v-else class="panel-empty">{{ search ? '没有匹配的商品' : '暂无商品' }}</div>
      </div>
    </section>

    <el-dialog :title="categoryDraft.id ? '修改分类名称' : '新增分类'" :visible.sync="categoryDialog" width="420px" custom-class="category-editor" append-to-body>
      <el-form label-position="top" @submit.native.prevent="saveCategoryName">
        <el-form-item label="分类名称" required>
          <el-input v-model.trim="categoryDraft.name" maxlength="60" placeholder="例如：包包、项链" />
        </el-form-item>
        <div class="dialog-actions">
          <el-button @click="categoryDialog = false">取消</el-button>
          <el-button type="primary" native-type="submit" :loading="categoryWorking" :disabled="!categoryDraft.name">保存分类</el-button>
        </div>
      </el-form>
    </el-dialog>

    <el-dialog :title="draft.id === 'new' ? '添加商品' : '编辑商品'" :visible.sync="dialog" width="720px"
      :close-on-click-modal="false" :close-on-press-escape="!saving" :show-close="!saving"
      append-to-body custom-class="product-editor" @closed="clearPhotos">
      <el-form id="product-editor-form" label-position="top" :disabled="saving" @submit.native.prevent="save">
        <el-form-item label="商品名称" required><el-input v-model.trim="draft.name" maxlength="30" /></el-form-item>
        <fieldset class="editor-categories" :disabled="saving">
          <legend>分类 <span aria-hidden="true">*</span></legend>
          <label v-for="category in categories" :key="category.id">
            <input v-model="draft.categoryId" type="radio" name="product-category" :value="category.id">
            <span>{{ category.name }}</span>
          </label>
        </fieldset>
        <div class="form-two">
          <el-form-item label="售价（元）" required><el-input v-model.trim="draft.yuan" inputmode="decimal" placeholder="例如 29.90" /></el-form-item>
          <el-form-item label="上架状态"><el-switch v-model="draft.available" active-text="在售" inactive-text="下架" /></el-form-item>
        </div>
        <section class="product-photos" aria-labelledby="product-photos-title">
          <h3 id="product-photos-title">商品相册</h3>
          <p class="form-hint">第一张作为商品列表封面。每组最多 8 张，单张不超过 4 MB。</p>
          <ProductPhotoGroup :key="editorSession" :photos="galleryPhotos" label="商品相册" primary-label="封面" :disabled="saving"
            @files="addPhotos(galleryPhotos, $event)" @link="addPhotoLink(galleryPhotos, $event)"
            @remove="removePhoto(galleryPhotos, $event)" @primary="makePrimary(galleryPhotos, $event)" />
        </section>
        <div class="sku-heading"><h3>规格与库存</h3><el-button type="text" icon="el-icon-plus" @click="addSKU">添加规格</el-button></div>
        <div class="editor-skus">
          <div v-for="(sku, index) in draft.skus" :key="index" class="editor-sku">
            <label>规格 / 戒围<input v-model.trim="sku.variant" :disabled="saving" class="el-input__inner" placeholder="标准款" maxlength="30"></label>
            <label>颜色<input v-model.trim="sku.color" :disabled="saving" @focus="originalColor = sku.color" @change="renameColorPhotos(sku.color)" class="el-input__inner" placeholder="原色" maxlength="30"></label>
            <label>库存<input v-model.trim="sku.stock" :disabled="saving" class="el-input__inner" name="sku-stock" inputmode="numeric" maxlength="6" placeholder="0"></label>
            <el-button type="text" class="danger-text" :disabled="draft.skus.length <= 1" :aria-label="'移除规格 ' + (index + 1)"
              @click="draft.skus.splice(index, 1)">移除</el-button>
          </div>
        </div>
        <p class="form-hint">每种规格和颜色单独计库存。</p>
        <section class="product-photos" aria-labelledby="color-photos-title">
          <h3 id="color-photos-title">按颜色配图</h3>
          <p class="form-hint">颜色来自上方规格。同色规格共用一组图，未配图时展示商品相册；没有相册时展示列表封面。</p>
          <details v-for="color in draftColors" :key="editorSession + ':' + color" class="color-photo-group">
            <summary><strong>{{ color }}</strong><span>{{ photosFor(color).length ? photosFor(color).length + ' 张' : '未配图' }}</span></summary>
            <ProductPhotoGroup :photos="photosFor(color)" :label="color" :disabled="saving"
              @files="addPhotos(photosFor(color, true), $event)" @link="addPhotoLink(photosFor(color, true), $event)"
              @remove="removePhoto(photosFor(color), $event)" @primary="makePrimary(photosFor(color), $event)" />
          </details>
          <p v-if="!draftColors.length" class="form-hint">填写规格颜色后，即可为该颜色添加图片。</p>
        </section>
        <el-collapse class="optional-product-fields">
          <el-collapse-item title="描述、材质与其他资料（选填）" name="details">
            <el-form-item label="商品描述"><el-input v-model.trim="draft.description" maxlength="120" /></el-form-item>
            <div class="form-two">
              <el-form-item label="英文名称"><el-input v-model.trim="draft.english" maxlength="60" /></el-form-item>
              <el-form-item label="材质"><el-input v-model.trim="draft.material" maxlength="100" /></el-form-item>
            </div>
            <el-form-item label="商品细节"><el-input v-model="draft.details" type="textarea" :rows="3" maxlength="800" /></el-form-item>
            <el-form-item label="养护说明"><el-input v-model="draft.care" type="textarea" :rows="3" maxlength="800" /></el-form-item>
          </el-collapse-item>
        </el-collapse>
      </el-form>
      <div slot="footer" class="dialog-actions product-editor-actions">
        <el-button :disabled="saving" @click="dialog = false">取消</el-button>
        <el-button type="primary" native-type="submit" form="product-editor-form" :loading="saving">保存商品</el-button>
      </div>
    </el-dialog>
  </div>
</template>

<script>
import { loadCatalog, saveProduct, saveCategory, deleteCategory, uploadImage } from '@/api/catalog'
import ProductPhotoGroup from '@/components/ProductPhotoGroup.vue'
import { money, parseYuan } from '@/utils/money'

const fresh = () => ({
  id: 'new', name: '', english: '', description: '', categoryId: null,
  yuan: '', image: '', material: '', details: '', care: '', available: true, version: 0,
  skus: [{ variant: '标准款', color: '原色', stock: 0 }],
})

const savedPhotos = urls => urls.map(url => ({ url, file: null }))

export default {
  components: { ProductPhotoGroup },
  data: () => ({
    products: [], categories: [], categoryId: null, search: '', loading: true,
    categoryDraft: { id: null, name: '' }, categoryDialog: false, categoryWorking: false,
    draft: fresh(), dialog: false, saving: false, workingProduct: '',
    galleryPhotos: [], colorPhotos: [], originalColor: '', editorSession: 0,
  }),
  computed: {
    draftColors() {
      return [...new Set(this.draft.skus.map(sku => sku.color.trim()).filter(Boolean))]
    },
    activeCategory() {
      return this.categories.find(category => category.id === this.categoryId)
    },
    categoryCounts() {
      const counts = {}
      for (const product of this.products) counts[product.categoryId] = (counts[product.categoryId] || 0) + 1
      return counts
    },
    filtered() {
      const query = this.search.trim().toLowerCase()
      return this.products.filter(product => (
        (!this.categoryId || product.categoryId === this.categoryId) &&
        (product.name + product.category).toLowerCase().includes(query)
      ))
    },
  },
  mounted() {
    this.load()
  },
  beforeDestroy() {
    this.clearPhotos()
  },
  methods: {
    money,
    async load() {
      this.loading = true
      try {
        const catalog = await loadCatalog()
        this.products = catalog.products
        this.categories = catalog.categories
        if (!this.activeCategory) this.categoryId = null
      } catch (error) {
        this.$message.error(error.message)
      } finally {
        this.loading = false
      }
    },
    editCategory(category) {
      this.categoryDraft = category ? { ...category } : { id: null, name: '' }
      this.categoryDialog = true
    },
    async saveCategoryName() {
      if (this.categoryWorking || !this.categoryDraft.name) return
      this.categoryWorking = true
      try {
        const category = await saveCategory(this.categoryDraft.id || 'new', { name: this.categoryDraft.name })
        this.categoryDialog = false
        await this.load()
        this.categoryId = category.id
        this.search = ''
        this.$message.success('分类已保存')
      } catch (error) {
        this.$message.error(error.message)
      } finally {
        this.categoryWorking = false
      }
    },
    async removeCategory(category) {
      if (this.categoryWorking) return
      try {
        await this.$confirm(
          `删除“${category.name}”会同时删除该分类下的全部商品（当前 ${this.categoryCounts[category.id] || 0} 件），此操作无法撤销。历史订单会保留。`,
          '删除分类及商品',
          { type: 'warning', confirmButtonText: '删除分类及商品', cancelButtonText: '取消' },
        )
      } catch {
        return
      }
      this.categoryWorking = true
      try {
        await deleteCategory(category.id)
        await this.load()
        this.$message.success('分类及商品已删除')
      } catch (error) {
        this.$message.error(error.message)
      } finally {
        this.categoryWorking = false
      }
    },
    edit(product) {
      this.clearPhotos()
      this.draft = product
        ? { ...product, yuan: money(product.price), skus: product.skus.map(sku => ({ ...sku })) }
        : { ...fresh(), categoryId: this.categoryId || (this.categories.length === 1 ? this.categories[0].id : null) }
      this.editorSession += 1
      if (product) {
        this.colorPhotos = Object.entries(product.colorImages || {}).map(([color, urls]) => ({ color, photos: savedPhotos(urls) }))
        this.galleryPhotos = savedPhotos(product.gallery?.length ? product.gallery :
          product.image && !this.colorPhotos.some(group => group.photos.length) ? [product.image] : [])
      }
      this.dialog = true
    },
    addSKU() {
      this.draft.skus.push({ variant: '', color: '', stock: 0 })
    },
    photosFor(color, create = false) {
      const group = this.colorPhotos.find(item => item.color === color)
      if (group) return group.photos
      const photos = []
      if (create) this.colorPhotos.push({ color, photos })
      return photos
    },
    renameColorPhotos(color) {
      const oldColor = this.originalColor
      if (!color || oldColor === color || this.draftColors.includes(oldColor) || this.photosFor(color).length) return
      const group = this.colorPhotos.find(item => item.color === oldColor)
      if (!group) return
      this.colorPhotos = this.colorPhotos.filter(item => item.color !== color)
      group.color = color
    },
    addPhotos(photos, files) {
      const formats = ['image/jpeg', 'image/png', 'image/webp', 'image/gif']
      if (files.some(file => !file.size || file.size > 4 * 1024 * 1024 || (file.type && !formats.includes(file.type)))) {
        this.$message.warning('请选择不超过 4 MB 的 JPG、PNG、WebP 或 GIF 图片')
        return
      }
      if (photos.length + files.length > 8) {
        this.$message.warning('每组最多 8 张图片，请减少选择数量')
        return
      }
      photos.push(...files.map(file => ({ file, url: URL.createObjectURL(file) })))
    },
    addPhotoLink(photos, link) {
      if (!/^https:\/\/[^\s]+$/i.test(link) || photos.some(photo => photo.url === link) || photos.length >= 8) {
        this.$message.warning('请输入有效且未添加的 HTTPS 图片地址')
        return
      }
      photos.push({ url: link, file: null })
    },
    removePhoto(photos, index) {
      const [photo] = photos.splice(index, 1)
      if (photo.file) URL.revokeObjectURL(photo.url)
    },
    makePrimary(photos, index) {
      const [photo] = photos.splice(index, 1)
      photos.unshift(photo)
    },
    clearPhotos() {
      for (const photo of [...this.galleryPhotos, ...this.colorPhotos.flatMap(group => group.photos)]) {
        if (photo.file) URL.revokeObjectURL(photo.url)
      }
      this.galleryPhotos = []
      this.colorPhotos = []
      this.originalColor = ''
    },
    async uploadPhotos(photos) {
      for (const photo of photos) {
        if (!photo.file) continue
        const { url } = await uploadImage(photo.file)
        URL.revokeObjectURL(photo.url)
        photo.url = url
        photo.file = null
      }
      return photos.map(photo => photo.url)
    },
    async save() {
      if (this.saving) return
      if (!this.draft.name || !this.draft.categoryId) {
        this.$message.warning('请填写商品名称并选择分类')
        return
      }
      if (this.draft.skus.some(({ stock }) => !/^\d{1,6}$/.test(String(stock)) || Number(stock) > 100000)) {
        this.$message.warning('库存须为 0 至 100000 的整数')
        return
      }
      if (!this.draft.skus.every(sku => sku.variant && sku.color)) {
        this.$message.warning('请填写每项规格和颜色')
        return
      }
      const colorGroups = this.colorPhotos.filter(group => this.draftColors.includes(group.color) && group.photos.length)
      if (!this.galleryPhotos.length && !colorGroups.length) {
        this.$message.warning('请在商品相册或任一颜色添加图片')
        return
      }
      this.saving = true
      try {
        const price = parseYuan(this.draft.yuan)
        const skus = this.draft.skus.map(sku => ({ ...sku, stock: Number(sku.stock) }))
        const gallery = await this.uploadPhotos(this.galleryPhotos)
        const colorImages = Object.create(null)
        for (const group of colorGroups) colorImages[group.color] = await this.uploadPhotos(group.photos)
        await saveProduct(this.draft.id, { ...this.draft, price, skus, image: gallery[0] || '', gallery, colorImages })
        this.dialog = false
        await this.load()
        this.$message.success('商品已保存')
      } catch (error) {
        this.$message.error(error.message)
      } finally {
        this.saving = false
      }
    },
    async toggle(product) {
      if (this.workingProduct) return
      this.workingProduct = product.id
      try {
        await saveProduct(product.id, { ...product, available: !product.available })
        await this.load()
        this.$message.success(product.available ? '商品已下架' : '商品已上架')
      } catch (error) {
        this.$message.error(error.message)
        await this.load()
      } finally {
        this.workingProduct = ''
      }
    },
  },
}
</script>
