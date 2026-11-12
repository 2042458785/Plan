<script setup lang="ts">
import { ref } from 'vue'
import Icon from '@/components/Icon.vue'
import { assetUrl, storefrontURL } from '@/config'
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
const defaultCover = assetUrl('/images/hero.jpg')
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
        <span class="quick-monogram">S</span>
        <span>{{ settings.name }}<small>快速小程序</small></span>
      </a>
      <div class="quick-actions">
        <a :href="storefrontURL" class="quick-full-shop">完整商城 <Icon name="arrow" :size="13" /></a>
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
      <section class="quick-cover" aria-labelledby="quick-title">
        <img :src="assetUrl(settings.quickCover || '/images/hero.jpg')" alt="" fetchpriority="high" @error="restoreCover" />
        <div class="quick-cover-shade"></div>
        <div class="quick-cover-copy">
          <span class="quick-kicker">SUXU / THE QUICK EDIT</span>
          <h1 id="quick-title">快速小程序</h1>
          <p>随手挑选，留一件心动。</p>
          <button type="button" class="quick-explore" @click="scrollToCatalog">
            开始选购 <Icon name="arrow" :size="16" />
          </button>
        </div>
        <span class="quick-cover-mark" aria-hidden="true">SUXU · 生活选物</span>
      </section>

      <div class="quick-notice">
        <span class="quick-shop-state" :class="{ open: settings.open }">
          <i aria-hidden="true"></i>{{ settings.open ? '接受预约' : '暂停预约' }}
        </span>
        <p v-if="settings.announcement">{{ settings.announcement }}</p>
      </div>

      <section ref="catalog" class="quick-catalog" :class="{ 'quick-catalog-empty': categories.length <= 1 }" aria-label="商品分类和列表">
        <nav v-if="categories.length > 1" class="quick-sidebar" aria-label="商品分类">
          <p class="quick-side-label">分类 <span>INDEX</span></p>
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
              <span class="quick-kicker">THE COLLECTION</span>
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
                <span v-if="product.english" class="quick-product-english">{{ product.english }}</span>
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
            <p>这里暂时没有商品。</p>
          </div>
        </div>
      </section>
    </main>

    <footer class="quick-footer">
      <span v-if="settings.location"><Icon name="pin" :size="16" />{{ settings.location }}</span>
      <span v-if="settings.hours"><Icon name="clock" :size="16" />{{ settings.hours }}</span>
      <span class="quick-footer-mark">SUXU · 自提与配送</span>
    </footer>
  </div>
</template>

<style scoped>
.quick-shop {
  --quick-ink: #34382b;
  --quick-muted: #6f6c5e;
  --quick-line: #e8e2d7;
  --quick-paper: #faf8f3;
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
  background: #faf8f3f5;
  backdrop-filter: blur(14px);
}

.quick-brand,
.quick-actions,
.quick-full-shop,
.quick-notice,
.quick-shop-state,
.quick-explore,
.quick-footer > span {
  display: flex;
  align-items: center;
}

.quick-brand { gap: 11px; min-width: 0; font-weight: 600; }
.quick-brand > span:last-child { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.quick-brand small { display: block; margin-top: 2px; color: var(--quick-muted); font-size: 11px; font-weight: 400; }
.quick-monogram {
  display: grid;
  place-items: center;
  flex: 0 0 37px;
  height: 37px;
  border: 1px solid #d8cfbd;
  border-radius: 50%;
  font: 24px Georgia, serif;
}

.quick-actions { flex-shrink: 0; gap: 6px; }
.quick-actions button { position: relative; display: grid; place-items: center; width: 44px; height: 44px; color: inherit; }
.quick-full-shop { gap: 6px; padding: 12px 6px; font-size: 12px; }
.quick-bag-count { position: absolute; top: 1px; right: 0; min-width: 18px; padding: 2px 4px; border-radius: 16px; background: #424636; color: #fff; font-size: 10px; }
.quick-main { max-width: 1160px; margin: auto; padding: 24px 20px 52px; }

.quick-cover {
  position: relative;
  isolation: isolate;
  display: flex;
  align-items: center;
  min-height: clamp(280px, 35vw, 380px);
  overflow: hidden;
  background: #77654f;
  color: #fffaf0;
}

.quick-cover > img,
.quick-cover-shade { position: absolute; inset: 0; width: 100%; height: 100%; }
.quick-cover > img { z-index: -2; object-fit: cover; object-position: 50% 50%; animation: quick-drift 4.8s ease-out both; }
.quick-cover-shade { z-index: -1; background: linear-gradient(92deg, #2b281fec, #3d34256b 54%, #241f1412 100%); }
.quick-cover-copy { padding: 32px clamp(27px, 6vw, 75px); animation: quick-reveal .7s ease-out both; }
.quick-kicker { display: block; font-size: 10px; font-weight: 600; letter-spacing: .17em; }
.quick-cover h1 { margin: 12px 0 10px; font: 400 clamp(32px, 5vw, 56px)/1.18 var(--quick-serif); letter-spacing: .07em; }
.quick-cover-copy p { font-size: 14px; letter-spacing: .07em; }
.quick-explore { gap: 15px; min-height: 43px; margin-top: 23px; padding: 0 15px; background: #f9f3e7; color: #403a2d; font-size: 12px; }
.quick-cover-mark { position: absolute; right: 24px; bottom: 20px; font-size: 10px; letter-spacing: .2em; }

.quick-notice { gap: 12px; padding: 15px 0; border-bottom: 1px solid var(--quick-line); color: var(--quick-muted); font-size: 12px; }
.quick-shop-state { gap: 7px; flex-shrink: 0; font-weight: 600; }
.quick-shop-state i { width: 7px; height: 7px; border-radius: 50%; background: #947250; }
.quick-shop-state.open i { background: #617746; }
.quick-notice p { min-width: 0; overflow-wrap: anywhere; }

.quick-catalog { display: grid; grid-template-columns: 164px minmax(0, 1fr); gap: 30px; margin-top: 24px; scroll-margin-top: 88px; }
.quick-sidebar { position: sticky; top: 88px; align-self: start; max-height: calc(100vh - 100px); overflow: auto; border-right: 1px solid var(--quick-line); }
.quick-side-label { padding: 3px 12px 10px; color: var(--quick-muted); font-size: 12px; }
.quick-side-label span { margin-left: 4px; font: 9px Georgia, serif; letter-spacing: .1em; }
.quick-sidebar button { display: block; width: 100%; min-height: 48px; padding: 9px 10px; border-left: 2px solid transparent; color: inherit; font-size: 14px; text-align: left; overflow-wrap: anywhere; }
.quick-sidebar button[aria-pressed="true"] { border-color: #515740; background: #ece8db; font-weight: 650; }
.quick-sidebar button:hover { background: #f0ece2; }
.quick-listing-heading { display: flex; justify-content: space-between; align-items: flex-end; gap: 12px; min-height: 56px; margin-bottom: 15px; }
.quick-listing-heading .quick-kicker, .quick-total { color: var(--quick-muted); }
.quick-listing-heading h2 { margin-top: 4px; font: 400 clamp(25px, 3vw, 34px)/1.2 var(--quick-serif); overflow-wrap: anywhere; }
.quick-total { padding-bottom: 2px; flex-shrink: 0; font-size: 12px; }
.quick-grid { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 16px; animation: quick-reveal .27s ease-out both; }
.quick-product { overflow: hidden; padding: 0; border: 1px solid var(--quick-line); background: #fffdf8; color: inherit; text-align: left; }
.quick-product-photo { position: relative; overflow: hidden; aspect-ratio: 1.19; background: #ece4d4; }
.quick-product-photo img { display: block; width: 100%; height: 100%; object-fit: cover; transition: transform .45s ease; }
.quick-product:hover img { transform: scale(1.04); }
.quick-product-tag { position: absolute; top: 8px; left: 8px; max-width: calc(100% - 16px); padding: 4px 7px; background: #fffaf0; color: #474335; font-size: 10px; overflow-wrap: anywhere; }
.quick-product-copy { padding: 11px 12px 10px; }
.quick-product-english { display: block; margin-bottom: 4px; color: var(--quick-muted); font-size: 9px; letter-spacing: .08em; overflow-wrap: anywhere; }
.quick-product h3 { font: 600 16px/1.35 var(--quick-serif); overflow-wrap: anywhere; }
.quick-product-bottom { display: flex; align-items: center; justify-content: space-between; gap: 6px; margin-top: 8px; font-size: 13px; }
.quick-product-arrow { display: grid; place-items: center; width: 28px; height: 28px; border-radius: 50%; background: #ede9de; }
.quick-empty { display: grid; justify-items: center; gap: 12px; padding: 48px 12px; border: 1px dashed var(--quick-line); color: var(--quick-muted); font-size: 13px; }
.quick-footer { display: flex; flex-wrap: wrap; gap: 12px 20px; max-width: 1120px; margin: auto; padding: 18px 20px 30px; border-top: 1px solid var(--quick-line); color: var(--quick-muted); font-size: 12px; }
.quick-footer > span { gap: 6px; }
.quick-footer-mark { margin-left: auto; letter-spacing: .12em; }

@keyframes quick-drift {
  from { transform: scale(1.01) translateX(0); }
  to { transform: scale(1.085) translateX(-1%); }
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
  .quick-brand { gap: 7px; font-size: 13px; }
  .quick-monogram { flex-basis: 32px; height: 32px; font-size: 21px; }
  .quick-actions { gap: 0; }
  .quick-full-shop { font-size: 11px; }
  .quick-full-shop svg { display: none; }
  .quick-main { padding: 10px 10px 32px; }
  .quick-cover { min-height: 250px; }
  .quick-cover > img { object-position: 57% center; }
  .quick-cover-shade { background: linear-gradient(94deg, #2b281fd9, #3d342554 78%); }
  .quick-cover-mark { right: 11px; bottom: 12px; }
  .quick-catalog { grid-template-columns: 85px minmax(0, 1fr); gap: 10px; margin-top: 15px; scroll-margin-top: 72px; }
  .quick-sidebar { top: 72px; max-height: calc(100vh - 78px); }
  .quick-side-label { padding-left: 7px; }
  .quick-side-label span { display: none; }
  .quick-sidebar button { padding: 8px 5px 8px 7px; font-size: 12px; }
  .quick-listing-heading { min-height: 46px; gap: 5px; margin-bottom: 10px; }
  .quick-listing-heading .quick-kicker { font-size: 8px; }
  .quick-total { font-size: 10px; }
  .quick-grid { grid-template-columns: minmax(0, 1fr); gap: 10px; }
  .quick-product-photo { aspect-ratio: 1.43; }
  .quick-footer { padding-inline: 10px; }
  .quick-footer-mark { margin-left: 0; }
}

@media (prefers-reduced-motion: reduce) {
  .quick-cover > img, .quick-cover-copy, .quick-grid { animation: none; }
  .quick-product-photo img { transition: none; }
}
.quick-catalog.quick-catalog-empty { grid-template-columns: minmax(0, 1fr); }
</style>
