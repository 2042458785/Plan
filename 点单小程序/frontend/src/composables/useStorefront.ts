import {ref,computed,onMounted,onUnmounted,watch,nextTick} from 'vue'
import store from '@/store'
import {shopApi as api} from '@/api'
import {randomKey} from '@/utils/request'
import type {Product,Category,ShopSetting,BagLine} from '@/types/catalog'

export function useStorefront(){

const products=ref<Product[]>([]),settings=ref<ShopSetting>({name:'素序 SUXU',announcement:'将日常，交给心动。',location:'',hours:'',open:false,version:0}),loading=ref(true),error=ref(''),toast=ref('')
const category=ref('全部商品'),sort=ref('curated'),search=ref(''),view=ref('home'),overlay=ref(''),variant=ref(''),color=ref(''),quantity=ref(1),zoomed=ref(false),accordion=ref('detail')
const selectedId = ref('')
const selected = computed(() => products.value.find(product => product.id === selectedId.value) ?? null)
const cart=computed<BagLine[]>({get:()=>store.state.bag.items,set:v=>store.commit('bag/setItems',v)}),favorites=computed<string[]>({get:()=>store.state.wishlist.ids,set:v=>store.commit('wishlist/setIds',v)}),history=ref(load('suxu-orders-v1',[])),historyData=ref([]),current=ref(null)
const customer=ref(''),contact=ref(''),note=ref(''),pickup=ref('营业时间内自提'),busy=ref(false),consent=ref(false),service=ref('about')
const fulfillment = ref<'pickup' | 'delivery'>('pickup'), address = ref(''), deliveryRate = ref(200)
const serviceReturn=ref('');const collection=ref<HTMLElement|null>(null),searchInput=ref<HTMLInputElement|null>(null),sheet=ref<HTMLElement|null>(null)
let interval,toastTimer;let focusBefore:HTMLElement|null=null
function load(k,d){try{return JSON.parse(localStorage.getItem(k))??d}catch{return d}}
function persist(k,v){try{k==='suxu-bag-v1'?store.commit('bag/setItems',JSON.parse(JSON.stringify(v))):k==='suxu-favorites-v1'?store.commit('wishlist/setIds',[...v]):localStorage.setItem(k,JSON.stringify(v))}catch{notify('浏览器存储不可用，请保留本次订单页面')}}
const money=n=>new Intl.NumberFormat('zh-CN',{maximumFractionDigits:2}).format(n/100)
const catalogCategories = ref<Category[]>([])
const listedProducts = computed(() => products.value.filter(product => product.available))
const categories = computed(() => ['全部商品', ...catalogCategories.value.map(item => item.name).filter(name => name !== '全部商品')])
const categoryInfo = computed(() => {
  const covers = new Map<string, string>()
  for (const product of listedProducts.value) {
    if (product.image && !covers.has(product.category)) covers.set(product.category, product.image)
  }
  return catalogCategories.value.filter(item => covers.has(item.name))
    .map(item => ({ ...item, image: covers.get(item.name) }))
})
const visible=computed(()=>{let data=listedProducts.value.filter(p=>(category.value==='全部商品'||p.category===category.value)&&(view.value!=='favorites'||favorites.value.includes(p.id))&&(!search.value||(p.name+p.description+p.category+p.english).toLowerCase().includes(search.value.toLowerCase())));if(sort.value==='low')data.sort((a,b)=>a.price-b.price);if(sort.value==='high')data.sort((a,b)=>b.price-a.price);return data})
const lines=computed(()=>cart.value.map(l=>{const p=products.value.find(p=>p.id===l.productId);return p?{...l,product:p}:null}).filter(Boolean))
const count = computed(() => lines.value.reduce((n, line) => n + line.quantity, 0))
const subtotal = computed(() => lines.value.reduce((n, line) => n + line.quantity * line.product.price, 0))
const deliveryFee = computed(() => fulfillment.value === 'delivery' ? deliveryRate.value : 0)
const total = computed(() => subtotal.value + deliveryFee.value)
const selectedSKU=computed(()=>selected.value?.skus?.find(s=>s.variant===variant.value&&s.color===color.value))
const unavailable=computed(()=>lines.value.some(l=>!l.product.available||(l.product.skus.find(s=>s.variant===l.variant&&s.color===l.color)?.stock||0)<l.quantity||!l.product.options?.includes(l.variant)||!l.product.colors?.includes(l.color)))
const statusLabels={pending:'待商家确认',confirmed:'备货中',ready:'待自提',completed:'已完成',cancelled:'已取消'}
function orderStatus(order) {
  return order.status === 'ready' && order.fulfillment === 'delivery' ? '待配送' : statusLabels[order.status]
}
watch(fulfillment, () => { consent.value = false }, { flush: 'sync' })
const year=new Date().getFullYear()
function notify(t){toast.value=t;clearTimeout(toastTimer);toastTimer=setTimeout(()=>toast.value='',3800)}

function updateTitle() {
  document.title = settings.value.name + ' · 日常之上，自有光芒'
}

async function refresh() {
  const catalog = await api('/catalog')
  const activeCategory = catalogCategories.value.find(item => item.name === category.value)
  catalogCategories.value = catalog.categories
  if (activeCategory) category.value = catalogCategories.value.find(item => item.id === activeCategory.id)?.name ?? '全部商品'
  products.value = catalog.products
  deliveryRate.value = catalog.deliveryFee ?? 200
  settings.value = catalog.settings
  if (!categories.value.includes(category.value)) category.value = '全部商品'
  if (overlay.value === 'product' && !selected.value) close()
  updateTitle()
  error.value = ''
}

function close(){if(overlay.value==='service'&&serviceReturn.value==='product'){overlay.value='product';serviceReturn.value='';return}overlay.value='';zoomed.value=false}
function open(name){overlay.value=name}
function openProduct(p){selectedId.value=p.id;variant.value=p.options?.length===1?p.options[0]:'';color.value=p.colors?.[0]||'';quantity.value=1;zoomed.value=false;accordion.value='detail';overlay.value='product'}
function goCollection(c='全部商品',v='home'){category.value=c;view.value=v;search.value='';close();nextTick(()=>collection.value?.scrollIntoView({behavior:matchMedia('(prefers-reduced-motion: reduce)').matches?'auto':'smooth',block:'start'}))}
function runSearch(){category.value='全部商品';view.value='home';close();nextTick(()=>collection.value?.scrollIntoView({behavior:'smooth'}))}
function home(){view.value='home';category.value='全部商品';search.value='';close();window.scrollTo({top:0,behavior:'smooth'})}
function favorite(p){favorites.value=favorites.value.includes(p.id)?favorites.value.filter(id=>id!==p.id):[...favorites.value,p.id];persist('suxu-favorites-v1',favorites.value);notify(favorites.value.includes(p.id)?'已加入心愿单':'已从心愿单移除')}
function addToBag(){const p=selected.value;if(!p?.available||p.stock<1||!settings.value.open){notify('这件商品暂不可预约');return}if(!variant.value){notify('请先选择尺寸 / 规格');return}if(!selectedSKU.value||selectedSKU.value.stock<1){notify('所选规格暂时售罄');return}if(!color.value){notify('请先选择颜色');return}const key=p.id+'|'+variant.value+'|'+color.value;const line=cart.value.find(l=>l.key===key);const existing=cart.value.filter(l=>l.key===key).reduce((s,l)=>s+l.quantity,0);if(existing+quantity.value>(selectedSKU.value?.stock||0)){notify('购物袋中的数量已超过当前库存');return}if(line){line.quantity+=quantity.value}else{cart.value.push({key,productId:p.id,variant:variant.value,color:color.value,quantity:quantity.value})}persist('suxu-bag-v1',cart.value);open('bag');notify('心选之物，已放入购物袋')}
function changeQuantity(l,d){const item=cart.value.find(v=>v.key===l.key);if(!item)return;const n=item.quantity+d;if(n<1){remove(l);return}const sum=cart.value.filter(v=>v.key===l.key).reduce((s,v)=>s+v.quantity,0);const sku=l.product.skus.find(s=>s.variant===l.variant&&s.color===l.color);if(d>0&&(sum>=(sku?.stock||0)||n>50)){notify('已达到当前可选数量');return}item.quantity=n;persist('suxu-bag-v1',cart.value)}
function remove(l){cart.value=cart.value.filter(v=>v.key!==l.key);persist('suxu-bag-v1',cart.value)}
async function submit() {
  if (busy.value || !lines.value.length) return
  if (!customer.value.trim()) { notify('请填写联系人称呼'); return }
  const delivering = fulfillment.value === 'delivery'
  if (delivering && (!contact.value.trim() || !address.value.trim())) {
    notify('请填写联系方式和配送地址')
    return
  }
  if (!consent.value) { notify('请确认取货和付款方式'); return }

  busy.value = true
  const payload = {
    items: lines.value.map(({ productId, quantity, variant, color }) => ({ productId, quantity, variant, color })),
    customer: customer.value,
    contact: contact.value,
    note: note.value,
    pickup: delivering ? '' : pickup.value,
    ...(delivering ? { fulfillment: 'delivery', address: address.value } : {}),
  }
  try {
    const signature = JSON.stringify(payload)
    let attempt = load('suxu-checkout-v1', null)
    if (!attempt || attempt.signature !== signature) {
      attempt = { key: randomKey(), signature }
      persist('suxu-checkout-v1', attempt)
    }
    const order = await api('/orders', { method: 'POST', body: JSON.stringify({ ...payload, key: attempt.key }) })
    history.value = [{ id: order.id, token: order.token }, ...history.value.filter(item => item.id !== order.id)].slice(0, 40)
    persist('suxu-orders-v1', history.value)
    current.value = order
    persist('suxu-bag-v1', [])
    persist('suxu-checkout-v1', null)
    consent.value = false
    note.value = ''
    open('success')
    notify('预约已提交')
  } catch (error) {
    notify(error.message)
  } finally {
    busy.value = false
  }
  await refresh().catch(() => {})
}
async function track(h){try{current.value=await api('/orders/'+h.id,{headers:{'X-Order-Token':h.token}});open('success')}catch(e){notify(e.message)}}
async function openOrders(){open('orders');historyData.value=[];const results=await Promise.allSettled(history.value.map(h=>api('/orders/'+h.id,{headers:{'X-Order-Token':h.token}})));historyData.value=results.filter(r=>r.status==='fulfilled').map(r=>r.value);if(results.some(r=>r.status==='rejected'))notify('部分订单暂时未能读取，请稍后重试')}
function showService(name){serviceReturn.value=overlay.value==='product'?'product':'';service.value=name;open('service')}
const serviceContent={about:{en:'THE WORLD OF SUXU',title:'素，是本真。序，是日常。',text:'我们偏爱有温度的织物、恰到好处的光泽，以及值得留在身边的小物。在这里，可以浏览商家上架的商品并提交预约。'},pickup:{en:'CLICK & COLLECT',title:'自提与送货上门',text:'下单时可选择自提或送货上门。自提请等商家备货后到店领取。送货上门需填写地址和联系方式，每单加收配送费，具体送达时间由商家确认。'},care:{en:'CARE & KEEP',title:'用心佩戴，长久相伴。',text:'织物建议局部轻柔清洁，避免暴晒。首饰避免接触香水、清洁剂，佩戴后擦拭并独立存放。莫桑石并非天然钻石，选购时请确认戒托材质、主石参数和戒围。实际材质与售后约定由主理人确认。'},size:{en:'FIND YOUR PERFECT FIT',title:'适合，才是恰好的美。',text:'戒指请选择你日常佩戴的港码戒围；不确定尺寸时，建议到店试戴并由主理人协助确认。布包与挂件可在自提时查看大小及搭配。具体规格请以商品详情为准。'}}
function keydown(e){if(e.key==='Escape')close();if(e.key==='Tab'&&overlay.value&&sheet.value){const targets=[...sheet.value.querySelectorAll('button:not(:disabled),a[href],input,select,textarea,[tabindex="0"]')].filter(n=>n.getClientRects().length);if(!targets.length)return;const first=targets[0] as HTMLElement,last=targets.at(-1) as HTMLElement;if(e.shiftKey&&document.activeElement===first){e.preventDefault();last.focus()}else if(!e.shiftKey&&document.activeElement===last){e.preventDefault();first.focus()}}}
watch(overlay,async(v,old)=>{document.body.style.overflow=v?'hidden':'';if(v&&!old)focusBefore=document.activeElement as HTMLElement;await nextTick();if(v){sheet.value?.scrollTo({top:0});sheet.value?.querySelector<HTMLElement>('button,input')?.focus({preventScroll:true});if(v==='search')searchInput.value?.focus()}else focusBefore?.focus?.({preventScroll:true})})
onMounted(async()=>{try{await refresh()}catch{error.value='暂时无法连接精品店，请稍后重试'}finally{loading.value=false}document.addEventListener('keydown',keydown);interval=setInterval(async()=>{try{await refresh();if(current.value&&overlay.value==='success'){current.value=await api('/orders/'+current.value.id,{headers:{'X-Order-Token':current.value.token}})}}catch{error.value='连接暂时中断，正在重试…'}},5000)})
onUnmounted(()=>{clearInterval(interval);clearTimeout(toastTimer);document.removeEventListener('keydown',keydown);document.body.style.overflow=''})
return {categoryInfo,categories,visible,products,settings,loading,error,toast,category,sort,search,view,overlay,selected,variant,color,quantity,zoomed,accordion,cart,favorites,history,historyData,current,customer,contact,note,pickup,busy,consent,service,fulfillment,address,deliveryRate,deliveryFee,orderStatus,collection,searchInput,sheet,lines,count,total,unavailable,selectedSKU,statusLabels,year,serviceContent,money,close,open,openProduct,goCollection,runSearch,home,favorite,addToBag,changeQuantity,remove,submit,track,openOrders,showService}
}
