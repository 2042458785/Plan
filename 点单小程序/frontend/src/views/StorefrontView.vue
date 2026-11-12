<script setup lang="ts">
import Icon from '@/components/Icon.vue'
import CustomerAccount from '@/components/CustomerAccount.vue'
import { quickShopURL } from '@/config'
import {useStorefront} from '@/composables/useStorefront'
const {categoryInfo,categories,visible,products,settings,loading,error,toast,category,sort,search,view,overlay,selected,variant,color,quantity,zoomed,accordion,bagProducts,detailInBag,legacySaved,productSections,historyData,historyLoading,historyError,current,account,accountSecure,accountLoading,authError,signIn,saveAccount,logout,checkout,setCheckout,pickupOptions,returnToCheckout,returnToBag,loadOrders,customer,contact,note,pickup,busy,consent,service,fulfillment,address,deliveryRate,deliveryFee,orderStatus,collection,searchInput,sheet,lines,count,subtotal,total,unavailable,selectedSKU,year,serviceContent,money,close,open,openProduct,goCollection,runSearch,home,bagHeart,addToBag,changeQuantity,remove,submit,track,openOrders,showService}=useStorefront()
</script>
<template>
<div class="boutique">
  <div v-if="loading" class="s-loading"><span>SUXU</span><small>美好，即将相见</small></div>
  <template v-else>
  <div v-if="error" class="s-connection" role="alert">{{ error }}</div>
  <div class="announcement-bar"><span>{{ settings.announcement }}</span><button @click="showService('pickup')">自提 · 送货上门 <Icon name="chevron" :size="11"/></button></div>
  <header class="s-header"><div class="header-left"><button class="icon-button menu-button" aria-label="打开系列导航" @click="open('navigation')"><Icon name="menu"/></button><button class="header-label" @click="showService('about')">独立生活选物 <span>EST. 2026</span></button></div><a class="s-logo" href="/" @click.prevent="home" aria-label="素序 SUXU 首页">SUXU<small>{{ settings.name?.split(' ')[0]||'素序' }} · 生活选物</small></a><div class="header-actions"><button class="icon-button" aria-label="搜索商品" @click="open('search')"><Icon name="search"/></button><button class="icon-button" aria-label="我的账号与预约" @click="openOrders()"><Icon name="user"/></button><button class="icon-button bag-button" aria-label="打开购物袋" @click="open('bag')"><Icon name="bag"/><span v-if="count" class="bag-count">{{ count }}</span></button></div></header>
  <nav class="s-nav" aria-label="商品系列"><button :class="{active:view==='home'&&category==='全部商品'}" @click="home">首页</button><button v-for="name in categories.slice(1)" :key="name" :class="{active:category===name}" @click="goCollection(name)">{{ name }}</button><a class="s-quick-link" :href="quickShopURL">快速小程序</a><button @click="showService('about')">关于素序</button></nav>
  <main>
    <section v-if="view==='home'&&category==='全部商品'&&!search" class="s-hero"><img class="hero-photograph" src="/images/hero.jpg" alt="素序系列：帆布包、指间光芒与精致挂件的静物大片" fetchpriority="high"/><div class="hero-shade"></div><div class="hero-editorial"><span class="s-eyebrow">THE ART OF EVERYDAY</span><h1>日常之上，<br>自有光芒。</h1><p>让触手可及的美好，<br>成为生活里恰好的注脚。</p><button class="s-text-link" @click="goCollection()">探索心选系列 <Icon name="arrow" :size="19"/></button></div><div class="hero-bottom"><span>SUXU · THE NEW EDIT</span><span>一件好物，一段相伴。</span><button @click="goCollection()" aria-label="向下探索商品">向下探索 <span>↓</span></button></div><div class="hero-side-note">OBJECTS OF QUIET BEAUTY</div></section>
    <section v-if="view==='home'&&category==='全部商品'&&!search&&categoryInfo.length" class="s-categories s-container"><div class="section-intro"><span class="s-eyebrow">A WORLD TO DISCOVER</span><h2>美好，各有其形。</h2><p>按分类浏览店内商品。</p></div><div class="category-grid"><button v-for="c in categoryInfo" :key="c.name" class="category-tile" @click="goCollection(c.name)"><div class="category-photo"><img :src="c.image" :alt="c.name" loading="lazy"/></div><div class="category-caption"><span>{{ c.name }}</span><Icon name="arrow" :size="19"/></div></button></div></section>
    <section ref="collection" class="s-collection s-container" id="collection"><div class="collection-heading"><div><span class="s-eyebrow">{{ category==='全部商品'?'THE CONSIDERED EDIT':'EXPLORE THE COLLECTION' }}</span><h2>{{ category==='全部商品'?'此刻，心动之选。':category }}</h2></div><span class="collection-count">{{ String(visible.length).padStart(2,'0') }} 件心选商品</span></div><div class="collection-toolbar"><div v-if="categories.length>1" class="collection-tabs"><button v-for="c in categories" :key="c" :class="{active:category===c}" @click="category=c">{{ c }}</button></div><label class="sort-label"><span>排序</span><select v-model="sort" aria-label="商品排序"><option value="curated">主理人精选</option><option value="low">价格从低到高</option><option value="high">价格从高到低</option></select></label></div><div v-if="search" class="search-context">“{{ search }}” 的搜索结果 <button @click="search=''">清除 <Icon name="close" :size="12"/></button></div><div class="s-product-grid"><article v-for="p in visible" :key="p.id" class="s-product"><div class="s-product-picture"><button class="product-open" :aria-label="'查看'+p.name" @click="openProduct(p)"><img :src="p.image" :alt="p.name" loading="lazy"/><span class="quick-view">探索商品 <Icon name="arrow" :size="17"/></span></button><button class="s-favorite" :class="{saved:bagProducts.has(p.id)}" :aria-label="(bagProducts.has(p.id)?'管理购物袋中的':'加入购物袋 ')+p.name" :aria-pressed="bagProducts.has(p.id)" @click="bagHeart(p)"><Icon name="heart" :size="18"/></button><span v-if="!p.available||p.stock<1" class="out-of-stock">暂时售罄</span></div><div class="s-product-info"><span class="product-english">{{ p.english }}</span><button @click="openProduct(p)"><h3>{{ p.name }}</h3></button><p>{{ p.description }}</p><div class="s-product-price"><span>¥ {{ money(p.price) }}</span><span class="product-color-count">{{ p.colors?.length||1 }} 色可选</span></div></div></article></div><div v-if="!visible.length" class="s-empty"><Icon name="search" :size="32"/><h3>还没遇见这件商品。</h3><p>试着换一个关键词，或探索其他系列。</p><button class="s-outline" @click="goCollection()">探索全部商品 <Icon name="arrow" :size="16"/></button></div><div class="collection-end"><span>THOUGHTFULLY CHOSEN. BEAUTIFULLY YOURS.</span><button @click="showService('pickup')">预约选购说明 <Icon name="arrow" :size="15"/></button></div></section>
    <template v-if="view==='home'&&category==='全部商品'&&!search">
    <section v-if="products.length" class="s-story"><div class="story-image"><img src="/images/ring.jpg" alt="六爪莫桑石戒指，指间的一束光" loading="lazy"/><span>THE LIGHT WITHIN · 02</span></div><div class="story-copy"><span class="s-eyebrow">A LITTLE LIGHT, A LASTING MOMENT</span><h2>不必盛大，<br>也能闪耀。</h2><p>一枚戒指，记录的从不只是某个时刻。<br>是每一天，依然愿意为自己心动。</p><span class="story-detail">莫桑钻戒系列 · 为日常留一束光</span><button class="s-text-link" @click="goCollection('莫桑钻戒')">发现指间光芒 <Icon name="arrow" :size="18"/></button></div></section>
    <section class="s-note s-container"><span class="s-eyebrow">LESS, BUT CLOSER TO YOU</span><h2>少一些喧嚣。<br>多一些，属于自己。</h2><p>我们相信，好看的物件，也应当好好陪伴生活。<br>在织物的触感、金属的弧度与微小的光芒里，找到自己的节奏。</p><button class="s-text-link" @click="showService('about')">走进素序 <Icon name="arrow" :size="17"/></button><span class="s-note-signature">Suxu, with love.</span></section>
    </template>
    <section class="s-services"><button @click="showService('pickup')"><Icon name="bag" :size="25"/><span>取货方式<small>自提或送货上门</small></span></button><button @click="showService('care')"><Icon name="diamond" :size="25"/><span>用心相伴<small>了解材质与日常养护</small></span></button></section>
  </main>
  <footer class="s-footer"><div class="footer-upper"><div class="footer-brand"><span>SUXU</span><small>素序 · 生活选物</small><p>好物有序，日常有光。</p></div><div class="footer-column"><h3>探索素序</h3><button v-for="name in categories.slice(1)" :key="name" @click="goCollection(name)">{{ name }}系列</button><a class="s-quick-link" :href="quickShopURL">快速小程序</a></div><div class="footer-column"><h3>客户关怀</h3><button @click="openOrders()">我的账号与预约</button><button @click="showService('size')">尺寸与选购</button><button @click="showService('care')">养护与售后</button></div><div class="footer-column footer-visit"><h3>相见于素序</h3><span>{{ settings.location }}</span><span>{{ settings.hours }}</span><button @click="showService('pickup')">查看取货方式 <Icon name="arrow" :size="15"/></button><span class="store-open"><i :class="{closed:!settings.open}"></i>{{ settings.open?'现正接受预约':'预约暂歇，欢迎浏览' }}</span></div></div><div class="footer-bottom"><span>© {{ year }} SUXU. ALL RIGHTS RESERVED.</span><span>独立审美 · 用心选物</span></div></footer>
  <nav class="mobile-bottom-nav" aria-label="快捷导航"><button :class="{active:view==='home'}" @click="home"><Icon name="diamond" :size="19"/><span>首页</span></button><button @click="goCollection()"><Icon name="menu" :size="19"/><span>系列</span></button><button @click="openOrders()"><Icon name="user" :size="19"/><span>我的</span></button><button @click="open('bag')"><span class="mobile-bag-icon"><Icon name="bag" :size="19"/><i v-if="count">{{ count }}</i></span><span>购物袋</span></button></nav>
  </template>
  <Transition name="veil"><div v-if="overlay" class="s-overlay" :class="{'product-overlay':overlay==='product'}" @click.self="close"><section ref="sheet" class="s-sheet" :class="{'s-product-sheet':overlay==='product','s-center-sheet':['service','success'].includes(overlay)}" role="dialog" aria-modal="true" :aria-label="{product:'商品详情',bag:'购物袋',search:'搜索商品',orders:'我的账号与预约',success:'预约详情',service:'客户服务',navigation:'系列导航'}[overlay]"><button class="sheet-close icon-button" aria-label="关闭窗口" @click="close"><Icon name="close" :size="22"/></button>
    <template v-if="overlay==='product'&&selected"><div class="detail-image" :class="{zoomed}"><img :src="selected.image" :alt="selected.name"/><button class="icon-button detail-zoom" :aria-label="zoomed?'缩小商品图':'放大商品图'" @click="zoomed=!zoomed"><Icon name="expand"/></button><span class="image-caption">SUXU · {{ selected.english }}</span></div><div class="detail-information"><span class="s-eyebrow">{{ selected.category }} / {{ selected.english }}</span><div class="detail-title"><h2>{{ selected.name }}</h2><button class="icon-button" :class="{saved:detailInBag}" :aria-label="(detailInBag?'管理购物袋中的':'加入购物袋 ')+selected.name"  :aria-pressed="detailInBag" @click="bagHeart(selected)"><Icon name="heart"/></button></div><p class="detail-description">{{ selected.description }}</p><strong class="detail-price">¥ {{ money(selected.price) }}</strong><div class="detail-color"><span>颜色 <b>{{ color||'请选择' }}</b></span><div class="color-options"><button v-for="c in selected.colors" :key="c" :class="{selected:color===c}" @click="color=c" :aria-label="'选择颜色'+c" :aria-pressed="color===c"><i :style="{background:({'奶油白':'#e6dfcf','燕麦色':'#cbb89a','墨黑':'#282824','银白色':'linear-gradient(135deg,#fff,#a5a5a5,#fff)','香槟金':'linear-gradient(135deg,#e0c99e,#b99a62,#e8d7ad)','象牙白':'#f1eadb','苔绿':'#6f7d5b'})[c]||'#c2b69f'}"></i></button></div></div><div class="size-heading"><span>尺寸 / 规格 <b v-if="variant">{{ variant }}</b></span><button @click="showService('size')">选购指南 <Icon name="arrow" :size="13"/></button></div><div class="variant-options"><button v-for="v in selected.options" :key="v" :class="{selected:variant===v}" :aria-pressed="variant===v" @click="variant=v">{{ v }}</button></div><div class="detail-quantity"><span>数量</span><div class="s-stepper"><button :disabled="quantity<=1" aria-label="减少购买数量" @click="quantity--"><Icon name="minus" :size="14"/></button><span>{{ quantity }}</span><button :disabled="quantity>=Math.min(selectedSKU?.stock||0,50)" aria-label="增加购买数量" @click="quantity++"><Icon name="plus" :size="14"/></button></div><small>{{ !variant?'请选择规格':selectedSKU?.stock>0?'可预约':'此规格暂时售罄' }}</small></div><button class="s-primary detail-add" :disabled="!selected.available||!selected.stock||selectedSKU?.stock===0||!settings.open" @click="addToBag">{{ !selected.available||!selected.stock?'暂时售罄':!settings.open?'暂停预约':'加入购物袋' }} <Icon name="bag" :size="19"/></button><p class="detail-assurance"><Icon name="check" :size="13"/>支持自提和送货上门</p><div class="detail-accordions"><div v-for="section in productSections" :key="section.key">
          <button type="button" :aria-expanded="accordion===section.key" @click="accordion=accordion===section.key?'':section.key">{{ section.name }}<Icon :name="accordion===section.key?'minus':'plus'" :size="15"/></button>
          <p v-if="accordion===section.key">{{ section.text }}</p>
        </div></div></div></template>
    <template v-else-if="overlay === 'bag'">
      <div class="sheet-heading">
        <h2 data-checkout-heading tabindex="-1">{{ checkout ? '填写预约' : '我的购物袋' }} <span v-if="!checkout">({{ count }})</span></h2>
        <p v-if="checkout">提交后由商家确认，取货时付款。</p>
      </div>
      <template v-if="count">
        <div v-if="!checkout" class="bag-scroll">
          <div v-for="l in lines" :key="l.key" class="s-bag-line">
            <button class="bag-line-image" @click="openProduct(l.product)"><img :src="l.product.image" :alt="l.product.name"></button>
            <div class="bag-line-description">
              <small v-if="l.product.english">{{ l.product.english }}</small>
              <h3>{{ l.product.name }}</h3>
              <p>{{ l.color }} · {{ l.variant }}</p>
              <strong>¥ {{ money(l.product.price) }}</strong>
              <div class="bag-line-actions">
                <div class="s-stepper">
                  <button :aria-label="'减少'+l.product.name" @click="changeQuantity(l,-1)"><Icon name="minus" :size="13"/></button>
                  <span>{{ l.quantity }}</span>
                  <button :aria-label="'增加'+l.product.name" @click="changeQuantity(l,1)"><Icon name="plus" :size="13"/></button>
                </div>
                <button class="remove-line" @click="remove(l)">移除</button>
              </div>
              <span v-if="!l.product.available||(l.product.skus.find(s=>s.variant===l.variant&&s.color===l.color)?.stock||0)<l.quantity" class="s-danger">所选规格库存不足，请调整数量或移除。</span>
            </div>
          </div>
        </div>
        <div v-else class="bag-scroll checkout-information">
          <button class="checkout-back" type="button" @click="setCheckout(false)">← 返回购物袋</button>
          <div v-if="account" class="checkout-account">
            <div><strong>{{ account.name }}</strong><span>{{ account.contact }} · {{ account.number }}</span></div>
            <button type="button" @click="openOrders(true)">编辑资料</button>
          </div>
          <template v-else>
            <button v-if="accountSecure" class="checkout-back" type="button" @click="openOrders(true)">登录后自动填写顾客资料</button>
            <div class="checkout-fields">
              <label>联系人称呼 <span>*</span><input v-model="customer" maxlength="30" placeholder="请填写称呼" autocomplete="name"></label>
              <label>联系方式 <span>*</span><input v-model="contact" maxlength="40" placeholder="手机号或微信号" autocomplete="off"></label>
            </div>
          </template>
          <fieldset class="fulfillment-options">
            <legend>取货方式</legend>
            <label><input v-model="fulfillment" type="radio" value="pickup"><span>自提</span></label>
            <label><input v-model="fulfillment" type="radio" value="delivery"><span>送货上门 <small>+¥ {{ money(deliveryRate) }}</small></span></label>
          </fieldset>
          <label v-if="fulfillment === 'pickup'">自提时间
            <select v-model="pickup"><option v-for="option in pickupOptions" :key="option">{{ option }}</option></select>
          </label>
          <label v-else>配送地址 <span>*</span>
            <input v-model="address" maxlength="100" placeholder="请填写寝室楼栋和寝室号，例如 3 栋 508" autocomplete="street-address">
          </label>
          <label>给主理人留言 <small>选填</small><textarea v-model="note" maxlength="180" rows="2" placeholder="尺寸咨询或其他需求"></textarea></label>
        </div>
      </template>
      <div v-else class="s-empty bag-empty">
        <Icon name="bag" :size="37"/><h3>购物袋是空的</h3>
        <button class="s-outline" @click="goCollection()">浏览商品 <Icon name="arrow" :size="17"/></button>
      </div>
      <section v-if="!checkout && legacySaved.length" class="legacy-saved">
        <h3>之前收藏的商品</h3>
        <button v-for="p in legacySaved" :key="p.id" type="button" @click="openProduct(p)">
          <img :src="p.image" :alt="p.name"><span>{{ p.name }}<small>选择规格后加入购物袋</small></span><Icon name="arrow" :size="16"/>
        </button>
      </section>
      <div v-if="count" class="bag-checkout" :class="{ 'is-checkout': checkout }" v-loading="busy" element-loading-background="rgba(255,254,250,0.8)">
        <div v-if="checkout && deliveryFee" class="bag-subtotal delivery-fee"><span>配送费</span><span>¥ {{ money(deliveryFee) }}</span></div>
        <div class="bag-subtotal"><span>{{ checkout ? '合计' : '商品小计' }} <small>{{ count }} 件商品</small></span><strong>¥ {{ money(checkout ? total : subtotal) }}</strong></div>
        <template v-if="checkout">
          <label class="consent-checkbox"><input v-model="consent" type="checkbox"><span>{{ fulfillment === 'delivery' ? '我已确认配送地址和费用。' : '我已确认自提安排。' }}</span></label>
          <button class="s-primary" :disabled="busy || unavailable || !settings.open" @click="submit">
            {{ busy ? '正在提交预约…' : !settings.open ? '店铺暂停预约' : unavailable ? '请调整购物袋中的商品' : '提交预约' }} <Icon name="arrow" :size="18"/>
          </button>
          <small class="no-charge-note">提交预约不会在线扣款</small>
        </template>
        <button v-else class="s-primary" :disabled="unavailable || !settings.open" @click="setCheckout(true)">
          {{ !settings.open ? '店铺暂停预约' : unavailable ? '请调整购物袋中的商品' : '继续填写预约' }} <Icon name="arrow" :size="18"/>
        </button>
      </div>
    </template>
    <template v-else-if="overlay==='search'"><div class="sheet-heading"><span class="s-eyebrow">FIND YOUR LITTLE TREASURE</span><h2>寻找你的心动。</h2></div><form class="s-search-form" @submit.prevent="runSearch"><Icon name="search" :size="21"/><input ref="searchInput" v-model="search" placeholder="搜索商品名称" aria-label="搜索商品"/><button type="submit" aria-label="查看搜索结果"><Icon name="arrow"/></button></form><div v-if="categories.length>1" class="search-suggestions"><span class="s-eyebrow">EXPLORE BY CATEGORY</span><button v-for="name in categories.slice(1)" :key="name" @click="goCollection(name)"><span>{{ name }}</span><Icon name="arrow" :size="18"/></button></div></template>
    <template v-else-if="overlay==='navigation'"><div class="sheet-heading"><span class="s-eyebrow">THE WORLD OF SUXU</span><h2>探索素序</h2></div><div class="navigation-links"><button @click="home">首页 <small>HOME</small><Icon name="arrow"/></button><button v-for="name in categories.slice(1)" :key="name" @click="goCollection(name)">{{ name }}<Icon name="arrow"/></button><a class="s-quick-link" :href="quickShopURL">快速小程序</a></div><div class="navigation-bottom">{{ settings.location }}<br>{{ settings.hours }}</div></template>
    <template v-else-if="overlay === 'orders'">
      <div class="sheet-heading">
        <h2>我的账号与预约</h2>
        <button v-if="returnToCheckout" class="checkout-back" type="button" @click="returnToBag">← 返回预约</button>
      </div>
      <CustomerAccount :account="account" :secure="accountSecure" :loading="accountLoading" :error="authError"
        @authenticate="signIn" @save="saveAccount" @logout="logout"/>
      <section class="account-orders" aria-label="我的预约">
        <h3>{{ account ? '我的预约' : '本机预约' }}</h3>
        <p v-if="historyLoading">正在读取预约…</p>
        <div v-else-if="historyError" role="alert"><p class="s-danger">{{ historyError }}</p><button class="s-outline" @click="loadOrders">重试</button></div>
        <p v-if="!historyLoading && !historyError && !historyData.length">暂无预约。</p>
        <div v-if="historyData.length" class="customer-order-list">
          <button v-for="o in historyData" :key="o.id" @click="track(o)">
            <div class="customer-order-heading"><span>{{ o.number }}</span><b :class="o.status">{{ orderStatus(o) }}</b></div>
            <div class="customer-order-images"><img v-for="(i,n) in o.items.slice(0,3)" :key="n" :src="i.image" :alt="i.name"></div>
            <span class="customer-order-foot">{{ new Date(o.createdAt).toLocaleDateString('zh-CN') }}<strong>¥ {{ money(o.total) }} <Icon name="chevron" :size="15"/></strong></span>
          </button>
        </div>
      </section>
    </template>
    <template v-else-if="overlay==='success'&&current"><div class="success-symbol"><Icon :name="current.status==='cancelled'?'close':'check'" :size="26"/></div><span class="s-eyebrow">A LITTLE BEAUTY, RESERVED FOR YOU</span><h2 class="success-title">{{ current.status==='ready'?'美好，已为你备妥。':current.status==='completed'?'谢谢你，让美好相伴。':current.status==='cancelled'?'本次预约已取消。':'心选已留，静候相见。' }}</h2><p class="success-subtitle">{{ orderStatus(current) }} · {{ current.paid?'已确认收款':current.fulfillment==='delivery'?'待线下付款':'到店付款' }}</p><div class="reservation-id"><span>预约号</span><strong>{{ current.number }}</strong><small v-if="current.customerNumber">顾客编号 {{ current.customerNumber }}</small></div><div v-if="current.status!=='cancelled'" class="reservation-steps"><span v-for="(s,i) in ['pending','confirmed','ready','completed']" :key="s" :class="{done:['pending','confirmed','ready','completed'].indexOf(current.status)>=i}"><i>{{ i+1 }}</i>{{ ['预约提交','确认备货',current.fulfillment==='delivery'?'送货上门':'到店自提','完成'][i] }}</span></div><div class="success-items"><div v-for="(i,n) in current.items" :key="n"><img :src="i.image" :alt="i.name"/><div><h3>{{ i.name }}</h3><p>{{ i.color }} · {{ i.variant }} × {{ i.quantity }}</p></div><span>¥ {{ money(i.price*i.quantity) }}</span></div></div><div v-if="current.deliveryFee" class="success-total delivery-fee"><span>配送费</span><span>¥ {{ money(current.deliveryFee) }}</span></div><div class="success-total"><span>合计</span><strong>¥ {{ money(current.total) }}</strong></div><div class="pickup-info"><Icon name="pin" :size="17"/><div v-if="current.fulfillment==='delivery'"><strong>送货上门</strong><p>{{ current.address }}</p></div><div v-else><strong>{{ settings.location || '自提' }}</strong><p>{{ [settings.hours,current.pickup].filter(Boolean).join(' · ') }}</p></div></div><p v-if="current.note" class="reservation-note">留言：{{ current.note }}</p><button class="s-primary" @click="close">继续探索 <Icon name="arrow" :size="18"/></button></template>
    <template v-else-if="overlay==='service'"><div class="service-illustration"><Icon :name="service==='pickup'?'bag':service==='care'?'diamond':'leaf'" :size="42"/></div><span class="s-eyebrow">{{ serviceContent[service].en }}</span><h2 class="service-title">{{ serviceContent[service].title }}</h2><p class="service-body">{{ serviceContent[service].text }}</p><div class="service-location"><Icon name="pin" :size="17"/><span>{{ settings.location }}<small>{{ settings.hours }}</small></span></div><button class="s-outline" @click="close">回到我的心选 <Icon name="arrow" :size="17"/></button></template>
  </section></div></Transition>
  <Transition name="toast"><div v-if="toast" class="s-toast" role="status"><Icon name="check" :size="16"/>{{ toast }}</div></Transition>
</div>
</template>
