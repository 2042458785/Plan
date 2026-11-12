<script setup lang="ts">
import { ref } from 'vue'
import Icon from '@/components/Icon.vue'
import { assetUrl } from '@/config'
import type { Product, ShopSetting } from '@/types/catalog'

defineProps<{
  categories: string[]
  category: string
  products: Product[]
  settings: ShopSetting
  count: number
}>()

const emit = defineEmits<{
  selectCategory: [name: string]
  viewProduct: [product: Product]
  openBag: []
  openOrders: []
}>()

const catalog = ref<HTMLElement | null>(null)
const defaultCover = assetUrl('/images/quick-cover.jpg')
function restoreCover(event: Event) {
  const image = event.currentTarget as HTMLImageElement
  if (image.getAttribute('src') !== defaultCover) image.src = defaultCover
}
const quickHome = import.meta.env.BASE_URL
const price = new Intl.NumberFormat('zh-CN', { maximumFractionDigits: 2 })

function scrollToCatalog() {
  const reducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)').matches
  catalog.value?.scrollIntoView({ behavior: reducedMotion ? 'auto' : 'smooth', block: 'start' })
}

function selectCategory(name: string) {
  emit('selectCategory', name)
  if (catalog.value && catalog.value.getBoundingClientRect().top < 0) scrollToCatalog()
}
</script>

<template>
  <div class="quick-shop">
    <header class="quick-header">
      <a class="quick-brand" :href="quickHome" aria-label="快速小程序首页">
        <span class="quick-monogram" aria-hidden="true">S</span>
        <h1 class="quick-wordmark">{{ settings.name }}<small>快速小程序</small></h1>
      </a>
      <div class="quick-actions">
        <button type="button" aria-label="我的账号与预约" @click="emit('openOrders')">
          <Icon name="user" :size="20" />
        </button>
        <button type="button" class="quick-bag" aria-label="打开购物袋" @click="emit('openBag')">
          <Icon name="bag" :size="21" />
          <span v-if="count" class="quick-bag-count">{{ count }}</span>
        </button>
      </div>
    </header>

    <main class="quick-main">
      <div class="quick-cover">
        <img :src="settings.quickCover ? assetUrl(settings.quickCover) : defaultCover"
          alt="" fetchpriority="high" @error="restoreCover" />
      </div>

      <div class="quick-notice">
        <span class="quick-shop-state" :class="{ open: settings.open }">
          <i aria-hidden="true"></i>{{ settings.open ? '接受预约' : '暂停预约' }}
        </span>
        <p v-if="settings.announcement">{{ settings.announcement }}</p>
      </div>

      <section ref="catalog" class="quick-catalog" :class="{ 'quick-catalog-empty': categories.length <= 1 }" aria-label="商品分类和列表">
        <nav v-if="categories.length > 1" class="quick-sidebar" aria-label="商品分类">
          <p class="quick-side-label">分类</p>
          <button
            v-for="name in categories"
            :key="name"
            type="button"
            :aria-pressed="category === name"
            @click="selectCategory(name)"
          >
            {{ name }}
          </button>
        </nav>

        <div class="quick-listing">
          <div class="quick-listing-heading">
            <div>
              <h2>{{ category }}</h2>
            </div>
            <span class="quick-total" aria-live="polite">{{ products.length }} 件商品</span>
          </div>

          <div v-if="products.length" :key="category" class="quick-grid">
            <button
              v-for="product in products"
              :key="product.id"
              type="button"
              class="quick-product"
              :aria-label="'查看' + product.name"
              @click="emit('viewProduct', product)"
            >
              <div class="quick-product-photo">
                <img :src="assetUrl(product.image)" :alt="product.name" loading="lazy" decoding="async" />
                <span v-if="product.stock < 1" class="quick-product-tag">暂时售罄</span>

              </div>
              <div class="quick-product-copy">
                <h3>{{ product.name }}</h3>
                <div class="quick-product-bottom">
                  <span>¥ {{ price.format(product.price / 100) }}</span>
                  <span class="quick-product-arrow"><Icon name="arrow" :size="16" /></span>
                </div>
              </div>
            </button>
          </div>
          <div v-else class="quick-empty">
            <Icon name="box" :size="30" />
            <p>暂无商品</p>
          </div>
        </div>
      </section>
    </main>

    <footer class="quick-footer">
      <span v-if="settings.location"><Icon name="pin" :size="16" />{{ settings.location }}</span>
      <span v-if="settings.hours"><Icon name="clock" :size="16" />{{ settings.hours }}</span>
      <span class="quick-footer-mark">素序 SUXU</span>
    </footer>
  </div>
</template>

<style scoped>
.quick-shop {
  --quick-ink: #30392b;
  --quick-muted: #625e50;
  --quick-line: #ded8ca;
  --quick-paper: #faf8f2;
  --quick-serif: "Songti SC", "Noto Serif CJK SC", SimSun, serif;
  min-height: 100vh;
  background: var(--quick-paper);
  color: var(--quick-ink);
}

.quick-header {
  position: sticky;
  top: 0;
  z-index: 4;
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
  min-height: 72px;
  padding: 9px max(22px, calc((100% - 1120px) / 2));
  border-bottom: 1px solid var(--quick-line);
  background: #faf8f2f5;
  backdrop-filter: blur(14px);
}

.quick-brand,
.quick-actions,
.quick-notice,
.quick-shop-state,
.quick-footer > span {
  display: flex;
  align-items: center;
}

.quick-brand { gap: 11px; min-width: 0; }
.quick-wordmark {
  margin: 0;
  font: 600 19px/1.25 var(--quick-serif);
  letter-spacing: .035em;
  overflow-wrap: anywhere;
}
.quick-wordmark small {
  display: block;
  margin-top: 3px;
  color: var(--quick-muted);
  font: 400 11px/1.25 system-ui, sans-serif;
  letter-spacing: .1em;
}
.quick-monogram {
  display: grid;
  place-items: center;
  flex: 0 0 37px;
  height: 37px;
  border: 1px solid #d8cfbd;
  border-radius: 50%;
  background: #eae5d8;
  font: 23px Georgia, serif;
}

.quick-actions { flex-shrink: 0; gap: 6px; }
.quick-actions button { position: relative; display: grid; place-items: center; width: 44px; height: 44px; color: inherit; }
.quick-bag-count { position: absolute; top: 1px; right: 0; min-width: 18px; padding: 2px 4px; border-radius: 16px; background: #424636; color: #fff; font-size: 10px; }
.quick-main { max-width: 1160px; margin: auto; padding: 24px 20px 52px; }

.quick-cover {
  position: relative;
  height: clamp(218px, 39vw, 465px);
  animation: quick-reveal .56s var(--motion-ease) both;
  overflow: hidden;
  border-radius: 18px;
  background: #ebe2d4;
}
.quick-cover > img { display: block; width: 100%; height: 100%; object-fit: cover; animation: quick-drift .9s var(--motion-ease) both; }

.quick-notice { gap: 12px; padding: 15px 0; border-bottom: 1px solid var(--quick-line); color: var(--quick-muted); font-size: 13px; }
.quick-shop-state { gap: 7px; flex-shrink: 0; font-weight: 600; }
.quick-shop-state i { width: 7px; height: 7px; border-radius: 50%; background: #947250; }
.quick-shop-state.open i { background: #617746; }
.quick-notice p { min-width: 0; overflow-wrap: anywhere; }

.quick-catalog { display: grid; grid-template-columns: 164px minmax(0, 1fr); gap: 30px; margin-top: 24px; scroll-margin-top: 88px; }
.quick-sidebar { position: sticky; top: 88px; align-self: start; max-height: calc(100vh - 100px); overflow: auto; border-right: 1px solid var(--quick-line); }
.quick-side-label { padding: 3px 12px 10px; color: var(--quick-muted); font-size: 12px; }
.quick-sidebar button { display: block; width: 100%; min-height: 48px; padding: 9px 10px; border-left: 2px solid transparent; border-radius: 0 8px 8px 0; color: inherit; font-size: 14px; text-align: left; overflow-wrap: anywhere; }
.quick-sidebar button[aria-pressed="true"] { border-color: #515740; background: #e7eadf; font-weight: 650; }
.quick-sidebar button:hover { background: #f0ece2; }
.quick-listing-heading { display: flex; justify-content: space-between; align-items: flex-end; gap: 12px; min-height: 56px; margin-bottom: 15px; }
.quick-total { color: var(--quick-muted); }
.quick-listing-heading h2 { margin-top: 4px; font: 400 clamp(25px, 3vw, 34px)/1.2 var(--quick-serif); overflow-wrap: anywhere; }
.quick-total { padding-bottom: 2px; flex-shrink: 0; font-size: 12px; }
.quick-grid { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 16px; animation: quick-reveal .3s var(--motion-ease) both; }
.quick-product { overflow: hidden; padding: 0; border: 1px solid var(--quick-line); border-radius: 12px; background: #fffdf8; color: inherit; text-align: left; }
.quick-product-photo { position: relative; overflow: hidden; aspect-ratio: 1.19; background: #ece4d4; }
.quick-product-photo img { display: block; width: 100%; height: 100%; object-fit: cover; transition: transform .45s ease; }
@media (hover: hover) { .quick-product:hover img { transform: scale(1.035); } }
.quick-product-tag { position: absolute; top: 8px; left: 8px; max-width: calc(100% - 16px); padding: 4px 7px; background: #fffaf0; color: #474335; font-size: 10px; overflow-wrap: anywhere; }
.quick-product-copy { padding: 11px 12px 10px; }
.quick-product h3 { font: 600 17px/1.4 var(--quick-serif); overflow-wrap: anywhere; }
.quick-product-bottom { display: flex; align-items: center; justify-content: space-between; gap: 6px; margin-top: 8px; font-size: 15px; font-variant-numeric: tabular-nums; }
.quick-product-arrow { display: grid; place-items: center; width: 28px; height: 28px; border-radius: 50%; background: #ede9de; }
.quick-empty { display: grid; justify-items: center; gap: 12px; padding: 48px 12px; border: 1px dashed var(--quick-line); color: var(--quick-muted); font-size: 13px; }
.quick-footer { display: flex; flex-wrap: wrap; gap: 12px 20px; max-width: 1120px; margin: auto; padding: 18px 20px 30px; border-top: 1px solid var(--quick-line); color: var(--quick-muted); font-size: 12px; }
.quick-footer > span { gap: 6px; }
.quick-footer-mark { margin-left: auto; letter-spacing: .12em; }

@keyframes quick-drift {
  from { transform: scale(1.025); }
  to { transform: scale(1); }
}

@keyframes quick-reveal {
  from { opacity: 0; transform: translateY(7px); }
  to { opacity: 1; transform: translateY(0); }
}

@media (max-width: 820px) {
  .quick-grid { grid-template-columns: repeat(2, minmax(0, 1fr)); }
  .quick-catalog { grid-template-columns: 126px minmax(0, 1fr); gap: 18px; }
}

@media (max-width: 560px) {
  .quick-header { min-height: 62px; padding: 6px 10px; }
  .quick-brand { gap: 9px; }
  .quick-wordmark { font-size: 16px; }
  .quick-monogram { flex-basis: 32px; height: 32px; font-size: 21px; }
  .quick-actions { gap: 0; }
  .quick-main { padding: 10px 10px 32px; }
  .quick-cover { height: auto; aspect-ratio: 1.54; border-radius: 13px; }
  .quick-cover > img { position: absolute; inset: 0; }
  .quick-catalog { grid-template-columns: 85px minmax(0, 1fr); gap: 10px; margin-top: 15px; scroll-margin-top: 72px; }
  .quick-sidebar { top: 72px; max-height: calc(100vh - 78px); }
  .quick-side-label { padding-left: 7px; }
  .quick-sidebar button { padding: 8px 5px 8px 7px; font-size: 12px; }
  .quick-listing-heading { min-height: 46px; gap: 5px; margin-bottom: 10px; }
  .quick-total { font-size: 10px; }
  .quick-grid { grid-template-columns: minmax(0, 1fr); gap: 10px; }
  .quick-product-photo { aspect-ratio: 1.43; }
  .quick-footer { padding-inline: 10px; }
  .quick-footer-mark { margin-left: 0; }
}

@media (prefers-reduced-motion: reduce) {
  .quick-cover, .quick-cover > img, .quick-grid { animation: none; }
  .quick-product-photo img { transition: none; }
}
.quick-catalog.quick-catalog-empty { grid-template-columns: minmax(0, 1fr); }
</style>
