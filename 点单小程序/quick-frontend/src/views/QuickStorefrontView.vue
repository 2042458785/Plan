<script setup lang="ts">
import Icon from '@/components/Icon.vue'
import QuickShop from '@/components/QuickShop.vue'
import { assetUrl } from '@/config'
import {useStorefront} from '@/composables/useStorefront'
const {categories,visible,settings,loading,error,toast,category,overlay,selected,variant,color,quantity,zoomed,accordion,bagProducts,detailInBag,legacySaved,history,historyData,current,checkout,setCheckout,pickupOptions,customer,contact,note,pickup,busy,consent,service,fulfillment,address,deliveryRate,deliveryFee,orderStatus,sheet,lines,count,subtotal,total,unavailable,selectedSKU,year,serviceContent,money,close,open,openProduct,goCollection,bagHeart,addToBag,changeQuantity,remove,submit,track,openOrders,showService}=useStorefront()
</script>
<template>
<div class="boutique">
  <div v-if="loading" class="s-loading"><span>SUXU</span><small>美好，即将相见</small></div>
  <template v-else>
  <div v-if="error" class="s-connection" role="alert">{{ error }}</div>
  <QuickShop
    :categories="categories"
    :category="category"
    :products="visible"
    :settings="settings"
    :count="count"
    @select-category="goCollection"
    @view-product="openProduct"
    @open-bag="open('bag')"
    @open-orders="openOrders"
  />
  </template>
  <Transition name="veil"><div v-if="overlay" class="s-overlay" :class="{'product-overlay':overlay==='product'}" @click.self="close"><section ref="sheet" class="s-sheet" :class="{'s-product-sheet':overlay==='product','s-center-sheet':['service','success'].includes(overlay)}" role="dialog" aria-modal="true" :aria-label="{product:'商品详情',bag:'购物袋',orders:'我的预约',success:'预约详情',service:'客户服务'}[overlay]"><button class="sheet-close icon-button" aria-label="关闭窗口" @click="close"><Icon name="close" :size="22"/></button>
    <template v-if="overlay==='product'&&selected"><div class="detail-image" :class="{zoomed}"><img :src="assetUrl(selected.image)" :alt="selected.name"/><button class="icon-button detail-zoom" :aria-label="zoomed?'缩小商品图':'放大商品图'" @click="zoomed=!zoomed"><Icon name="expand"/></button><span class="image-caption">SUXU · {{ selected.english }}</span></div><div class="detail-information"><span class="s-eyebrow">{{ selected.category }} / {{ selected.english }}</span><div class="detail-title"><h2>{{ selected.name }}</h2><button class="icon-button" :class="{saved:detailInBag}" :aria-label="(detailInBag?'管理购物袋中的':'加入购物袋 ')+selected.name"  :aria-pressed="detailInBag" @click="bagHeart(selected)"><Icon name="heart"/></button></div><p class="detail-description">{{ selected.description }}</p><strong class="detail-price">¥ {{ money(selected.price) }}</strong><div class="detail-color"><span>颜色 <b>{{ color||'请选择' }}</b></span><div class="color-options"><button v-for="c in selected.colors" :key="c" :class="{selected:color===c}" @click="color=c" :aria-label="'选择颜色'+c" :aria-pressed="color===c"><i :style="{background:({'奶油白':'#e6dfcf','燕麦色':'#cbb89a','墨黑':'#282824','银白色':'linear-gradient(135deg,#fff,#a5a5a5,#fff)','香槟金':'linear-gradient(135deg,#e0c99e,#b99a62,#e8d7ad)','象牙白':'#f1eadb','苔绿':'#6f7d5b'})[c]||'#c2b69f'}"></i></button></div></div><div class="size-heading"><span>尺寸 / 规格 <b v-if="variant">{{ variant }}</b></span><button @click="showService('size')">选购指南 <Icon name="arrow" :size="13"/></button></div><div class="variant-options"><button v-for="v in selected.options" :key="v" :class="{selected:variant===v}" :aria-pressed="variant===v" @click="variant=v">{{ v }}</button></div><div class="detail-quantity"><span>数量</span><div class="s-stepper"><button :disabled="quantity<=1" aria-label="减少购买数量" @click="quantity--"><Icon name="minus" :size="14"/></button><span>{{ quantity }}</span><button :disabled="quantity>=Math.min(selectedSKU?.stock||0,50)" aria-label="增加购买数量" @click="quantity++"><Icon name="plus" :size="14"/></button></div><small>{{ !variant?'请选择规格':selectedSKU?.stock>0?'可预约':'此规格暂时售罄' }}</small></div><button class="s-primary detail-add" :disabled="!selected.available||!selected.stock||selectedSKU?.stock===0||!settings.open" @click="addToBag">{{ !selected.available||!selected.stock?'暂时售罄':!settings.open?'暂停预约':'加入购物袋' }} <Icon name="bag" :size="19"/></button><p class="detail-assurance"><Icon name="check" :size="13"/>支持自提和送货上门</p><div class="detail-accordions"><div v-for="a in [{key:'detail',name:'商品细节'},{key:'care',name:'材质与养护'},{key:'delivery',name:'取货方式'}]" :key="a.key"><button @click="accordion=accordion===a.key?'':a.key">{{ a.name }}<Icon :name="accordion===a.key?'minus':'plus'" :size="15"/></button><p v-if="accordion===a.key">{{ a.key==='detail'?selected.details:a.key==='care'?selected.material+'。建议温和清洁、独立收纳，具体养护和售后约定请向主理人确认。':'提交预约时可选择自提或送货上门，商家确认后安排备货。' }}</p></div></div></div></template>
    <template v-else-if="overlay === 'bag'">
      <div class="sheet-heading">
        <h2 data-checkout-heading tabindex="-1">{{ checkout ? '填写预约' : '我的购物袋' }} <span v-if="!checkout">({{ count }})</span></h2>
        <p v-if="checkout">提交后由商家确认，取货时付款。</p>
      </div>
      <template v-if="count">
        <div v-if="!checkout" class="bag-scroll">
          <div v-for="l in lines" :key="l.key" class="s-bag-line">
            <button class="bag-line-image" @click="openProduct(l.product)"><img :src="assetUrl(l.product.image)" :alt="l.product.name"></button>
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
          <div class="checkout-fields">
            <label>联系人称呼 <span>*</span><input v-model="customer" maxlength="30" placeholder="请填写称呼" autocomplete="name"></label>
            <label>联系方式 <span>*</span><input v-model="contact" maxlength="40" placeholder="手机号或微信号" autocomplete="off"></label>
          </div>
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
          <img :src="assetUrl(p.image)" :alt="p.name"><span>{{ p.name }}<small>选择规格后加入购物袋</small></span><Icon name="arrow" :size="16"/>
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
    <template v-else-if="overlay==='orders'"><div class="sheet-heading"><span class="s-eyebrow">YOUR SUXU MOMENTS</span><h2>我的预约</h2><p>每一份心选，都值得期待。</p></div><div v-if="!history.length" class="s-empty"><Icon name="box" :size="35"/><h3>美好，从一次心动开始。</h3><p>提交预约后，可在这里查看备货和取货状态。</p><button class="s-outline" @click="goCollection()">发现商品 <Icon name="arrow" :size="16"/></button></div><div v-else class="customer-order-list"><button v-for="o in historyData" :key="o.id" @click="track(o)"><div class="customer-order-heading"><span>{{ o.number }}</span><b :class="o.status">{{ orderStatus(o) }}</b></div><div class="customer-order-images"><img v-for="(i,n) in o.items.slice(0,3)" :key="n" :src="assetUrl(i.image)" :alt="i.name"/></div><span class="customer-order-foot">{{ new Date(o.createdAt).toLocaleDateString('zh-CN') }}<strong>¥ {{ money(o.total) }} <Icon name="chevron" :size="15"/></strong></span></button></div><p class="storage-note">预约入口保存在当前浏览器，请保留预约编号。</p></template>
    <template v-else-if="overlay==='success'&&current"><div class="success-symbol"><Icon :name="current.status==='cancelled'?'close':'check'" :size="26"/></div><span class="s-eyebrow">A LITTLE BEAUTY, RESERVED FOR YOU</span><h2 class="success-title">{{ current.status==='ready'?'美好，已为你备妥。':current.status==='completed'?'谢谢你，让美好相伴。':current.status==='cancelled'?'本次预约已取消。':'心选已留，静候相见。' }}</h2><p class="success-subtitle">{{ orderStatus(current) }} · {{ current.paid?'已确认收款':current.fulfillment==='delivery'?'待线下付款':'到店付款' }}</p><div class="reservation-id"><span>预约号</span><strong>{{ current.number }}</strong></div><div v-if="current.status!=='cancelled'" class="reservation-steps"><span v-for="(s,i) in ['pending','confirmed','ready','completed']" :key="s" :class="{done:['pending','confirmed','ready','completed'].indexOf(current.status)>=i}"><i>{{ i+1 }}</i>{{ ['预约提交','确认备货',current.fulfillment==='delivery'?'送货上门':'到店自提','完成'][i] }}</span></div><div class="success-items"><div v-for="(i,n) in current.items" :key="n"><img :src="assetUrl(i.image)" :alt="i.name"/><div><h3>{{ i.name }}</h3><p>{{ i.color }} · {{ i.variant }} × {{ i.quantity }}</p></div><span>¥ {{ money(i.price*i.quantity) }}</span></div></div><div v-if="current.deliveryFee" class="success-total delivery-fee"><span>配送费</span><span>¥ {{ money(current.deliveryFee) }}</span></div><div class="success-total"><span>合计</span><strong>¥ {{ money(current.total) }}</strong></div><div class="pickup-info"><Icon name="pin" :size="17"/><div v-if="current.fulfillment==='delivery'"><strong>送货上门</strong><p>{{ current.address }}</p></div><div v-else><strong>{{ settings.location || '自提' }}</strong><p>{{ [settings.hours,current.pickup].filter(Boolean).join(' · ') }}</p></div></div><p v-if="current.note" class="reservation-note">留言：{{ current.note }}</p><button class="s-primary" @click="close">继续探索 <Icon name="arrow" :size="18"/></button></template>
    <template v-else-if="overlay==='service'"><div class="service-illustration"><Icon :name="service==='pickup'?'bag':service==='care'?'diamond':'leaf'" :size="42"/></div><span class="s-eyebrow">{{ serviceContent[service].en }}</span><h2 class="service-title">{{ serviceContent[service].title }}</h2><p class="service-body">{{ serviceContent[service].text }}</p><div class="service-location"><Icon name="pin" :size="17"/><span>{{ settings.location }}<small>{{ settings.hours }}</small></span></div><button class="s-outline" @click="close">回到我的心选 <Icon name="arrow" :size="17"/></button></template>
  </section></div></Transition>
  <Transition name="toast"><div v-if="toast" class="s-toast" role="status"><Icon name="check" :size="16"/>{{ toast }}</div></Transition>
</div>
</template>
