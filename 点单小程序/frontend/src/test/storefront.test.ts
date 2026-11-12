import {mount,flushPromises} from '@vue/test-utils'
import {beforeEach,afterEach,describe,it,expect,vi} from 'vitest'
import {readFileSync} from 'node:fs'
import {resolve} from 'node:path'
import StorefrontView from '@/views/StorefrontView.vue'
import store from '@/store'
import {shopApi} from '@/api'
vi.mock('@/api',()=>({shopApi:vi.fn()}))
const seed=JSON.parse(readFileSync(resolve(process.cwd(),'src/test/catalog.fixture.json'),'utf8'))
const makeCatalog=()=>({deliveryFee:200,categories:[...new Set<string>(seed.products.map((p:any)=>p.category))].map((name,i)=>({id:i+1,name,position:i})),settings:{...seed.settings,version:1},products:seed.products.map((p:any)=>{const skus=[];let id=1;for(const variant of p.options)for(const color of p.colors)skus.push({id:id++,productId:p.id,variant,color,stock:Math.floor(p.stock/(p.options.length*p.colors.length)),active:true,version:1});return {...p,version:1,skus,stock:skus.reduce((n,s)=>n+s.stock,0)}})})
let wrapper:any
const api=vi.mocked(shopApi)
function button(label:string){const node=wrapper.findAll('button').find((b:any)=>b.text().trim()===label);expect(node,`找不到按钮 ${label}`).toBeTruthy();return node}
beforeEach(()=>{localStorage.clear();store.commit('bag/setItems',[]);store.commit('wishlist/setIds',[]);vi.stubGlobal('matchMedia',()=>({matches:true}));window.scrollTo=vi.fn();Element.prototype.scrollIntoView=vi.fn();Element.prototype.scrollTo=vi.fn();api.mockImplementation(async(path:string,options:any={})=>{if(path==='/catalog')return makeCatalog();if(path==='/orders'){const body=JSON.parse(options.body);return {id:'fixture-order',number:'SU-TEST-001',token:'fixture-private-order-token',status:'pending',paid:false,deliveryFee:body.fulfillment==='delivery'?200:0,total:body.items.reduce((n:any,i:any)=>n+69900*i.quantity,body.fulfillment==='delivery'?200:0),createdAt:new Date().toISOString(),version:1,...body,items:body.items.map((i:any)=>({...i,name:'初光 · 经典莫桑钻戒',price:69900,image:'/images/ring.jpg'}))}}throw Error('未声明的测试请求')})})
afterEach(()=>{wrapper?.unmount();wrapper=null;vi.clearAllMocks();vi.useRealTimers()})
async function render(){wrapper=mount(StorefrontView,{global:{plugins:[store],directives:{loading:{}}}});await flushPromises();return wrapper}
describe('顾客端行为与隔离',()=>{
 it('目录有四类商品，页面不含商家入口或管理组件',async()=>{await render();expect(wrapper.findAll('.s-product')).toHaveLength(7);expect(wrapper.find('a[href*="admin"]').exists()).toBe(false);expect(wrapper.text()).not.toContain('商家入口');expect(wrapper.find('.atelier').exists()).toBe(false)})
 it('戒围必选，库存按所选SKU限制，购物袋保留规格',async()=>{await render();await wrapper.get('button[aria-label="查看初光 · 经典莫桑钻戒"]').trigger('click');await button('加入购物袋').trigger('click');expect(wrapper.find('.s-bag-line').exists()).toBe(false);await button('港码 12').trigger('click');await wrapper.get('button[aria-label="增加购买数量"]').trigger('click');await wrapper.get('button[aria-label="增加购买数量"]').trigger('click');expect(wrapper.get('button[aria-label="增加购买数量"]').attributes('disabled')).toBeDefined();await button('加入购物袋').trigger('click');await flushPromises();expect(wrapper.get('.s-bag-line').text()).toContain('港码 12');expect(store.state.bag.items[0].quantity).toBe(3);expect(store.state.bag.items[0].color).toBe('银白色')})
 it('购物袋确认后提交规格，并保存预约查询凭证',async()=>{await render();await wrapper.get('button[aria-label="查看初光 · 经典莫桑钻戒"]').trigger('click');await button('港码 14').trigger('click');await button('加入购物袋').trigger('click');await button('继续填写预约').trigger('click');await wrapper.get('input[autocomplete="name"]').setValue('组件验收');await wrapper.get('input[placeholder="手机号或微信号"]').setValue('fixture-contact');await wrapper.get('.consent-checkbox input').setValue(true);await button('提交预约').trigger('click');await flushPromises();const request=api.mock.calls.find(c=>c[0]==='/orders');expect(request).toBeTruthy();const payload=JSON.parse(request![1].body);expect(payload.items[0]).toMatchObject({productId:'p2',variant:'港码 14',color:'银白色',quantity:1});expect(payload.key.length).toBeGreaterThanOrEqual(16);expect(wrapper.text()).toContain('SU-TEST-001');expect(JSON.parse(localStorage.getItem('suxu-orders-v1')||'[]')[0].token).toBe('fixture-private-order-token');expect(store.state.bag.items).toEqual([])})
 it('单规格商品的爱心和购物袋操作同一份清单',async()=>{
   await render()
   const product = makeCatalog().products.find(p => p.skus.length === 1)
   await wrapper.get(`button[aria-label="加入购物袋 ${product.name}"]`).trigger('click')
   expect(store.state.bag.items[0].productId).toBe(product.id)
   expect(store.state.wishlist.ids).toEqual([])
   await wrapper.get('button[aria-label="打开购物袋"]').trigger('click')
   expect(wrapper.get('.s-bag-line').text()).toContain(product.name)
   expect(wrapper.find('.checkout-information').exists()).toBe(false)
   await wrapper.get('button[aria-label="关闭窗口"]').trigger('click')
   await wrapper.get(`button[aria-label="管理购物袋中的${product.name}"]`).trigger('click')
   expect(store.state.bag.items).toEqual([])
 })
 it('选购指南返回后保留已选戒围',async()=>{await render();await wrapper.get('button[aria-label="查看初光 · 经典莫桑钻戒"]').trigger('click');await button('港码 16').trigger('click');await button('选购指南').trigger('click');expect(wrapper.text()).toContain('适合，才是恰好的美。');await button('回到我的心选').trigger('click');expect(button('港码 16').attributes('aria-pressed')).toBe('true')})
})

describe('目录同步', () => {
  it('刷新后使用新价格、下架状态和预约开关', async () => {
    vi.useFakeTimers()
    const catalog = makeCatalog()
    api.mockResolvedValueOnce(catalog)
    await render()
    await wrapper.get('button[aria-label="查看初光 · 经典莫桑钻戒"]').trigger('click')
    const updated = makeCatalog()
    updated.products[0].available = false
    updated.products[1].price = 76900
    updated.settings.open = false
    api.mockResolvedValueOnce(updated)

    await vi.advanceTimersByTimeAsync(5000)
    await flushPromises()
    expect(wrapper.get('.detail-price').text()).toContain('769')
    expect(wrapper.get('.detail-add').attributes('disabled')).toBeDefined()
    expect(wrapper.find('button[aria-label="查看云游 · 弧形帆布包"]').exists()).toBe(false)
  })
})

it('送货上门需要地址及联系方式，整单加 2 元', async () => {
  await render()
  await wrapper.get('button[aria-label="查看初光 · 经典莫桑钻戒"]').trigger('click')
  await button('港码 14').trigger('click')
  await button('加入购物袋').trigger('click')
  await button('继续填写预约').trigger('click')
  await wrapper.get('input[autocomplete="name"]').setValue('配送测试')
  await wrapper.get('input[value="delivery"]').setValue(true)
  expect(wrapper.get('.bag-checkout .delivery-fee').text()).toContain('¥ 2')
  expect(wrapper.get('.bag-checkout .bag-subtotal:last-of-type').text()).toContain('701')
  await wrapper.get('.consent-checkbox input').setValue(true)
  await button('提交预约').trigger('click')
  expect(api.mock.calls.some(c => c[0] === '/orders')).toBe(false)
  await wrapper.get('input[placeholder="手机号或微信号"]').setValue('test_wechat_01')
  await wrapper.get('input[autocomplete="street-address"]').setValue('配送测试地址')
  await button('提交预约').trigger('click')
  await flushPromises()
  const request = api.mock.calls.find(c => c[0] === '/orders')!
  expect(JSON.parse(request[1].body)).toMatchObject({fulfillment:'delivery', address:'配送测试地址', pickup:''})
  expect(wrapper.get('.pickup-info').text()).toContain('配送测试地址')
})

it('商品细节和材质养护分别显示商家填写的内容', async () => {
  const catalog = makeCatalog()
  catalog.products[1] = {...catalog.products[1], details:'商品细节第一行\n第二行', material:'银色戒托', care:'单独收纳\n避免摩擦'}
  api.mockResolvedValueOnce(catalog)
  await render()
  await wrapper.get('button[aria-label="查看初光 · 经典莫桑钻戒"]').trigger('click')
  expect(wrapper.get('.detail-accordions p').element.textContent).toBe('商品细节第一行\n第二行')
  await button('材质与养护').trigger('click')
  expect(wrapper.get('.detail-accordions p').element.textContent).toBe('银色戒托\n\n单独收纳\n避免摩擦')
})
