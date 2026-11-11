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
      <el-table v-loading="loading" :data="filtered" stripe :empty-text="search ? '没有匹配的商品' : '暂无商品'">
        <el-table-column label="商品" min-width="220">
          <template slot-scope="{row}">
            <div class="table-artwork">
              <img :src="row.image" :alt="row.name" loading="lazy">
              <div><strong>{{ row.name }}</strong><span>{{ row.category }}</span></div>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="售价" width="100">
          <template slot-scope="{row}">¥ {{ money(row.price) }}</template>
        </el-table-column>
        <el-table-column label="可售库存" prop="stock" width="85" />
        <el-table-column label="状态" width="85">
          <template slot-scope="{row}">
            <el-tag size="small" :type="row.available ? 'success' : 'info'" effect="plain">{{ row.available ? '在售' : '已下架' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="130">
          <template slot-scope="{row}">
            <el-button type="text" @click="edit(row)">编辑</el-button>
            <el-button type="text" :disabled="!!workingProduct" @click="toggle(row)">{{ row.available ? '下架' : '上架' }}</el-button>
          </template>
        </el-table-column>
      </el-table>
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
      :close-on-click-modal="false" append-to-body custom-class="product-editor" @closed="clearImageSelection">
      <el-form label-position="top" @submit.native.prevent="save">
        <div class="form-two">
          <el-form-item label="商品名称" required><el-input v-model.trim="draft.name" maxlength="30" /></el-form-item>
          <el-form-item label="分类" required>
            <el-select v-model="draft.categoryId" filterable placeholder="选择已有分类">
              <el-option v-for="c in categories" :key="c.id" :label="c.name" :value="c.id" />
            </el-select>
          </el-form-item>
        </div>
        <div class="form-two">
          <el-form-item label="售价（元）" required><el-input v-model.trim="draft.yuan" inputmode="decimal" placeholder="例如 29.90" /></el-form-item>
          <el-form-item label="上架状态"><el-switch v-model="draft.available" active-text="在售" inactive-text="下架" /></el-form-item>
        </div>
        <el-form-item label="商品图片" required>
          <div class="product-image-control">
            <img v-if="imagePreview || draft.image" :src="imagePreview || draft.image" alt="商品图片预览">
            <div>
              <input ref="fileInput" class="product-file-input" type="file" accept="image/jpeg,image/png,image/webp,image/gif" @change="selectImage">
              <el-button size="small" @click="$refs.fileInput.click()">选择图片</el-button>
              <el-button v-if="imageFile" type="text" @click="clearImageSelection">取消选择</el-button>
              <p class="form-hint">{{ imageFile ? imageFile.name : 'JPG、PNG、WebP、GIF，最多 4 MB。' }}</p>
            </div>
          </div>
          <el-input v-if="!imageFile" v-model.trim="draft.image" maxlength="1000" placeholder="也可以填写 HTTPS 图片地址" />
        </el-form-item>
        <div class="sku-heading"><h3>规格与库存</h3><el-button type="text" icon="el-icon-plus" @click="addSKU">添加规格</el-button></div>
        <el-table :data="draft.skus" border size="small">
          <el-table-column label="规格 / 戒围" min-width="100">
            <template slot-scope="{row}"><el-input v-model.trim="row.variant" placeholder="标准款" maxlength="30" /></template>
          </el-table-column>
          <el-table-column label="颜色" min-width="90">
            <template slot-scope="{row}"><el-input v-model.trim="row.color" placeholder="原色" maxlength="30" /></template>
          </el-table-column>
          <el-table-column label="库存" width="115">
            <template slot-scope="{row}"><el-input-number v-model="row.stock" :min="0" :max="100000" controls-position="right" size="small" /></template>
          </el-table-column>
          <el-table-column width="55">
            <template slot-scope="scope">
              <el-button type="text" class="danger-text" :disabled="draft.skus.length <= 1" @click="draft.skus.splice(scope.$index, 1)">移除</el-button>
            </template>
          </el-table-column>
        </el-table>
        <p class="form-hint">每种规格和颜色单独计库存。</p>
        <el-collapse class="optional-product-fields">
          <el-collapse-item title="描述、材质与其他资料（选填）" name="details">
            <el-form-item label="商品描述"><el-input v-model.trim="draft.description" maxlength="120" /></el-form-item>
            <div class="form-two">
              <el-form-item label="英文名称"><el-input v-model.trim="draft.english" maxlength="60" /></el-form-item>
              <el-form-item label="材质"><el-input v-model.trim="draft.material" maxlength="100" /></el-form-item>
            </div>
            <el-form-item label="商品详情与养护"><el-input v-model.trim="draft.details" type="textarea" :rows="3" maxlength="800" /></el-form-item>
          </el-collapse-item>
        </el-collapse>
        <div class="dialog-actions">
          <el-button @click="dialog = false">取消</el-button>
          <el-button type="primary" native-type="submit" :loading="saving">保存商品</el-button>
        </div>
      </el-form>
    </el-dialog>
  </div>
</template>

<script>
import { loadCatalog, saveProduct, saveCategory, deleteCategory, uploadImage } from '@/api/catalog'
import { money, parseYuan } from '@/utils/money'

const fresh = () => ({
  id: 'new', name: '', english: '', description: '', categoryId: null,
  yuan: '', image: '', material: '', details: '', available: true, version: 0,
  skus: [{ variant: '标准款', color: '原色', stock: 0 }],
})

export default {
  data: () => ({
    products: [], categories: [], categoryId: null, search: '', loading: true,
    categoryDraft: { id: null, name: '' }, categoryDialog: false, categoryWorking: false,
    draft: fresh(), dialog: false, saving: false, workingProduct: '',
    imageFile: null, imagePreview: '',
  }),
  computed: {
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
    this.clearImageSelection()
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
      this.clearImageSelection()
      this.draft = product
        ? { ...product, yuan: money(product.price), skus: product.skus.map(sku => ({ ...sku })) }
        : { ...fresh(), categoryId: this.categoryId || (this.categories.length === 1 ? this.categories[0].id : null) }
      this.dialog = true
    },
    addSKU() {
      this.draft.skus.push({ variant: '', color: '', stock: 0 })
    },
    selectImage(event) {
      const file = event.target.files[0]
      if (!file) return
      if (file.size > 4 * 1024 * 1024 || (file.type && !['image/jpeg', 'image/png', 'image/webp', 'image/gif'].includes(file.type))) {
        this.clearImageSelection()
        this.$message.error('请选择不超过 4 MB 的 JPG、PNG、WebP 或 GIF 图片')
        return
      }
      if (this.imagePreview) URL.revokeObjectURL(this.imagePreview)
      this.imageFile = file
      this.imagePreview = URL.createObjectURL(file)
    },
    clearImageSelection() {
      if (this.imagePreview) URL.revokeObjectURL(this.imagePreview)
      this.imageFile = null
      this.imagePreview = ''
      if (this.$refs.fileInput) this.$refs.fileInput.value = ''
    },
    async save() {
      if (this.saving) return
      if (!this.draft.name || !this.draft.categoryId || !(this.imageFile || this.draft.image)) {
        this.$message.warning('请填写商品名称，选择分类和图片')
        return
      }
      this.saving = true
      try {
        const price = parseYuan(this.draft.yuan)
        if (this.imageFile) {
          this.draft.image = (await uploadImage(this.imageFile)).url
          this.clearImageSelection()
        }
        await saveProduct(this.draft.id, { ...this.draft, price })
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
