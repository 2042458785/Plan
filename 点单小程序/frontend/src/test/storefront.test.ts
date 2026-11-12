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
beforeEach(()=>{localStorage.clear();store.commit('bag/setItems',[]);store.commit('wishlist/setIds',[]);vi.stubGlobal('matchMedia',()=>({matches:true}));window.scrollTo=vi.fn();Element.prototype.scrollIntoView=vi.fn();Element.prototype.scrollTo=vi.fn();api.mockImplementation(async(path:string,options:any={})=>{if(path==='/catalog')return makeCatalog();if(path==='/orders'){const body=JSON.parse(options.body);const itemProduct=(i:any)=>seed.products.find(p=>p.id===i.productId);return {id:'fixture-order',number:'SU-TEST-001',token:'fixture-private-order-token',status:'pending',paid:false,deliveryFee:body.fulfillment==='delivery'?200:0,total:body.items.reduce((n:any,i:any)=>n+itemProduct(i).price*i.quantity,body.fulfillment==='delivery'?200:0),createdAt:new Date().toISOString(),version:1,...body,items:body.items.map((i:any)=>({...i,name:itemProduct(i).name,price:itemProduct(i).price,image:itemProduct(i).image}))}}throw Error('未声明的测试请求')})})
afterEach(()=>{wrapper?.unmount();wrapper=null;vi.clearAllMocks();vi.useRealTimers()})
async function render(){wrapper=mount(StorefrontView,{global:{plugins:[store],directives:{loading:{}}}});await flushPromises();return wrapper}
describe('顾客端行为与隔离',()=>{
 it('目录有四类商品，页面不含商家入口或管理组件',async()=>{await render();expect(wrapper.findAll('.s-product')).toHaveLength(7);expect(wrapper.find('a[href*="admin"]').exists()).toBe(false);expect(wrapper.text()).not.toContain('商家入口');expect(wrapper.find('.atelier').exists()).toBe(false)})
 it('立即购买先选择规格，最多可选到库存数量，并可返回商品', async () => {
   await render()
   await wrapper.get('button[aria-label="查看初光 · 经典莫桑钻戒"]').trigger('click')
   await button('立即购买').trigger('click')
   expect(wrapper.find('.checkout-information').exists()).toBe(false)
   await button('港码 12').trigger('click')
   await wrapper.get('button[aria-label="增加购买数量"]').trigger('click')
   await wrapper.get('button[aria-label="增加购买数量"]').trigger('click')
   expect(wrapper.get('button[aria-label="增加购买数量"]').attributes('disabled')).toBeDefined()
   await button('立即购买').trigger('click')
   expect(wrapper.get('.checkout-summary').text()).toContain('港码 12 · 银白色 × 3')
   expect(store.state.bag.items).toEqual([])
   await button('← 返回商品').trigger('click')
   expect(wrapper.find('.detail-information').exists()).toBe(true)
   expect(button('港码 12').attributes('aria-pressed')).toBe('true')
 })
 it('立即购买仅提交当前规格，购物袋中相同规格的商品也保留', async () => {
   const original = { key:'ring', productId:'p2', variant:'港码 14', color:'银白色', quantity:2 }
   store.commit('bag/setItems',[original])
   await render()
   await wrapper.get('button[aria-label="查看初光 · 经典莫桑钻戒"]').trigger('click')
   await button('港码 14').trigger('click')
   await button('立即购买').trigger('click')
   await wrapper.get('input[autocomplete="name"]').setValue('组件验收')
   await wrapper.get('input[placeholder="手机号或微信号"]').setValue('fixture-contact')
   await wrapper.get('.consent-checkbox input').setValue(true)
   await button('提交预约').trigger('click')
   await flushPromises()
   const request = api.mock.calls.find(c=>c[0]==='/orders')!
   const payload = JSON.parse(request[1].body)
   expect(payload.items).toEqual([{productId:'p2',variant:'港码 14',color:'银白色',quantity:1}])
   expect(payload.key.length).toBeGreaterThanOrEqual(16)
   expect(wrapper.text()).toContain('SU-TEST-001')
   expect(JSON.parse(localStorage.getItem('suxu-orders-v1')||'[]')[0].token).toBe('fixture-private-order-token')
   expect(store.state.bag.items).toEqual([original])
 })
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
  await button('立即购买').trigger('click')
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

it('仅结算勾选的行，未选和缺货商品继续留在购物袋', async () => {
  const catalog = makeCatalog()
  const products = [catalog.products[1], catalog.products[2], catalog.products[3]]
  products[2].skus[0].stock = 0
  const entries = products.map((product, index) => ({
    key: 'line-' + index, productId: product.id, variant: product.skus[0].variant, color: product.skus[0].color, quantity: 1,
  }))
  store.commit('bag/setItems', entries)
  api.mockResolvedValueOnce(catalog)
  await render()
  await wrapper.get('button[aria-label="打开购物袋"]').trigger('click')
  const selections = wrapper.findAll('.s-bag-line input[type="checkbox"]')
  const selectAll = wrapper.get('.bag-check-all input')
  expect(button('请勾选商品').attributes('disabled')).toBeDefined()
  expect(selections[2].attributes('disabled')).toBeDefined()
  await selectAll.setValue(true)
  expect(selections.map(item => item.element.checked)).toEqual([true,true,false])
  await selections[1].setValue(false)
  expect(selectAll.element.indeterminate).toBe(true)
  expect(wrapper.get('.bag-subtotal strong').text()).toContain('699')
  await button('继续填写预约').trigger('click')
  expect(wrapper.findAll('.checkout-summary li')).toHaveLength(1)
  await wrapper.get('input[autocomplete="name"]').setValue('组件验收')
  await wrapper.get('input[placeholder="手机号或微信号"]').setValue('fixture-contact')
  await wrapper.get('.pickup-options input[value="明日营业时间内自提"]').setValue(true)
  await wrapper.get('.consent-checkbox input').setValue(true)
  await button('提交预约').trigger('click')
  await flushPromises()
  const request = api.mock.calls.find(([path]) => path === '/orders')!
  expect(JSON.parse(request[1].body)).toMatchObject({
    items: [{productId:entries[0].productId, variant:entries[0].variant, color:entries[0].color, quantity:1}],
    pickup:'明日营业时间内自提',
  })
  expect(store.state.bag.items).toEqual(entries.slice(1))
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

it('提交失败保留商品及勾选，同一份预约可用原请求键重试', async () => {
  const entry = { key:'ring', productId:'p2', variant:'港码 12', color:'银白色', quantity:1 }
  store.commit('bag/setItems', [entry])
  const request = api.getMockImplementation()!
  let shouldFail = true
  api.mockImplementation((path, options) => {
    if (path === '/orders' && shouldFail) {
      shouldFail = false
      return Promise.reject(new Error('网络请求超时'))
    }
    return request(path, options)
  })
  await render()
  await wrapper.get('button[aria-label="打开购物袋"]').trigger('click')
  await wrapper.get('.s-bag-line input[type="checkbox"]').setValue(true)
  await button('继续填写预约').trigger('click')
  await wrapper.get('input[autocomplete="name"]').setValue('组件验收')
  await wrapper.get('input[placeholder="手机号或微信号"]').setValue('fixture-contact')
  await wrapper.get('.consent-checkbox input').setValue(true)
  await button('提交预约').trigger('click')
  await flushPromises()
  expect(wrapper.text()).toContain('网络请求超时')
  expect(store.state.bag.items).toEqual([entry])
  expect(wrapper.findAll('.checkout-summary li')).toHaveLength(1)
  await button('重试原预约').trigger('click')
  await flushPromises()
  const keys = api.mock.calls.filter(([path]) => path === '/orders').map(([,options]) => JSON.parse(options.body).key)
  expect(keys).toHaveLength(2)
  expect(keys[0]).toBe(keys[1])
  expect(store.state.bag.items).toEqual([])
})

it('详情页点爱心不会替换成另一个有货规格', async () => {
  const catalog = makeCatalog()
  const ring = catalog.products[1]
  ring.skus.forEach((sku, index) => { sku.stock = index === 0 ? 1 : 0 })
  ring.stock = 1
  api.mockResolvedValueOnce(catalog)
  await render()
  await wrapper.get('button[aria-label="查看初光 · 经典莫桑钻戒"]').trigger('click')
  await button('港码 12').trigger('click')
  await wrapper.get('.detail-title button').trigger('click')
  expect(store.state.bag.items).toEqual([])
  expect(button('港码 12').attributes('aria-pressed')).toBe('true')
  expect(wrapper.text()).toContain('暂不可预约')
})

it('当前颜色没有该规格时，保留购物袋中的其他规格', async () => {
  const catalog = makeCatalog()
  const ring = catalog.products[1]
  const color = ring.colors[0]
  ring.colors.push('另一个颜色')
  ring.skus = ring.skus.filter(sku => sku.variant !== '港码 12' || sku.color !== color)
  ring.skus.push({ ...ring.skus[0], id:100, variant:'港码 12', color:'另一个颜色' })
  const entry = {key:'ring', productId:ring.id, variant:'港码 10', color, quantity:1}
  store.commit('bag/setItems', [entry])
  api.mockResolvedValueOnce(catalog)
  await render()
  await wrapper.get('button[aria-label="查看初光 · 经典莫桑钻戒"]').trigger('click')
  await button('港码 12').trigger('click')
  expect(wrapper.get('.detail-title button').attributes('aria-pressed')).toBe('false')
  await wrapper.get('.detail-title button').trigger('click')
  expect(store.state.bag.items).toEqual([entry])
  expect(wrapper.text()).toContain('当前选择的颜色和规格无法购买')
})

it('选中 31 个规格时在购物袋提示上限，可以取消一项继续', async () => {
  const catalog = makeCatalog()
  const ring = catalog.products[1]
  ring.skus = Array.from({length:31}, (_, i) => ({ ...ring.skus[0], id:i+1, variant:'规格 '+(i+1), stock:1 }))
  ring.options = ring.skus.map(sku => sku.variant)
  ring.stock = ring.skus.length
  store.commit('bag/setItems', ring.skus.map(sku => ({key:String(sku.id),productId:ring.id,variant:sku.variant,color:sku.color,quantity:1})))
  api.mockResolvedValueOnce(catalog)
  await render()
  await wrapper.get('button[aria-label="打开购物袋"]').trigger('click')
  await wrapper.get('.bag-check-all input').setValue(true)
  expect(button('单次最多结算 30 个规格').attributes('disabled')).toBeDefined()
  await wrapper.findAll('.s-bag-line input[type="checkbox"]')[0].setValue(false)
  await button('继续填写预约').trigger('click')
  expect(wrapper.findAll('.checkout-summary li')).toHaveLength(30)
})

it('改价冲突后显示新金额，需重新确认并复用预约请求键', async () => {
  let catalog = makeCatalog()
  const originalRequest = api.getMockImplementation()!
  let changed = false
  api.mockImplementation((path, options) => {
    if (path === '/catalog') return Promise.resolve(catalog)
    if (path === '/orders' && !changed) {
      changed = true
      catalog = makeCatalog()
      catalog.products[1].price = 70900
      return Promise.reject(Object.assign(new Error('商品价格或配送费已变化，请重新确认金额'), {status:409}))
    }
    return originalRequest(path, options)
  })
  await render()
  await wrapper.get('button[aria-label="查看初光 · 经典莫桑钻戒"]').trigger('click')
  await button('港码 12').trigger('click')
  await button('立即购买').trigger('click')
  await wrapper.get('input[autocomplete="name"]').setValue('组件验收')
  await wrapper.get('input[placeholder="手机号或微信号"]').setValue('fixture-contact')
  await wrapper.get('.consent-checkbox input').setValue(true)
  await button('提交预约').trigger('click')
  await flushPromises()
  expect(wrapper.get('.bag-subtotal strong').text()).toBe('¥ 709')
  expect(wrapper.get('.consent-checkbox input').element.checked).toBe(false)
  await button('提交预约').trigger('click')
  expect(api.mock.calls.filter(([path]) => path === '/orders')).toHaveLength(1)
  await wrapper.get('.consent-checkbox input').setValue(true)
  await button('提交预约').trigger('click')
  await flushPromises()
  const [first, second] = api.mock.calls.filter(([path]) => path === '/orders').map(([, options]) => JSON.parse(options.body))
  expect(first.expectedTotal).toBe(69900)
  expect(second.expectedTotal).toBe(70900)
  expect(first.key).toBe(second.key)
})

it('最后一件已售出且原响应丢失时，仍可取回原预约并保留未选商品', async () => {
  const initial = makeCatalog()
  const ring = initial.products[1]
  ring.skus[0].stock = 1
  const chosen = {key:'ring', productId:ring.id, variant:ring.skus[0].variant, color:ring.skus[0].color, quantity:1}
  const other = initial.products[2]
  const unselected = {key:'other', productId:other.id, variant:other.skus[0].variant, color:other.skus[0].color, quantity:1}
  store.commit('bag/setItems', [chosen, unselected])
  const request = api.getMockImplementation()!
  let catalog = initial
  let acceptedOrder
  let originalBody: string
  api.mockImplementation(async (path, options) => {
    if (path === '/catalog') return catalog
    if (path === '/orders') {
      if (!acceptedOrder) {
        originalBody = options.body
        acceptedOrder = await request(path, options)
        const refreshed = makeCatalog()
        refreshed.products = refreshed.products.filter(product => product.id !== chosen.productId)
        refreshed.settings.open = false
        catalog = refreshed
        throw new Error('连接中断，未收到服务器响应')
      }
      expect(options.body).toBe(originalBody)
      return acceptedOrder
    }
    return request(path, options)
  })
  await render()
  await wrapper.get('button[aria-label="打开购物袋"]').trigger('click')
  await wrapper.findAll('.s-bag-line input[type="checkbox"]')[0].setValue(true)
  await button('继续填写预约').trigger('click')
  await wrapper.get('input[autocomplete="name"]').setValue('组件验收')
  await wrapper.get('input[placeholder="手机号或微信号"]').setValue('fixture-contact')
  await wrapper.get('.consent-checkbox input').setValue(true)
  await button('提交预约').trigger('click')
  await flushPromises()
  expect(wrapper.find('.checkout-summary').exists()).toBe(false)
  expect(button('重试原预约').attributes('disabled')).toBeUndefined()
  await button('重试原预约').trigger('click')
  await flushPromises()
  expect(wrapper.get('.reservation-id strong').text()).toBe('SU-TEST-001')
  expect(store.state.bag.items).toEqual([unselected])
})
