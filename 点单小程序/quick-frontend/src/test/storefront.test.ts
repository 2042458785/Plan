import {mount,flushPromises} from '@vue/test-utils'
import {beforeEach,afterEach,describe,it,expect,vi} from 'vitest'
import {readFileSync} from 'node:fs'
import {resolve} from 'node:path'
import QuickStorefrontView from '@/views/QuickStorefrontView.vue'
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
async function render(){wrapper=mount(QuickStorefrontView,{global:{plugins:[store],directives:{loading:{}}}});await flushPromises();return wrapper}
describe('快速小程序', () => {
  it('左侧分类来自目录，选定分类后通过原有预约流程加入购物袋', async () => {
    const catalog = makeCatalog()
    catalog.categories.find(c => c.name === catalog.products[1].category)!.name = '戒指专场'
    catalog.products[1].category = '戒指专场'
    api.mockResolvedValueOnce(catalog)
    await render()

    const categoryButton = wrapper.findAll('.quick-sidebar button').find(node => node.text() === '戒指专场')
    await categoryButton.trigger('click')
    expect(categoryButton.attributes('aria-pressed')).toBe('true')
    expect(wrapper.findAll('.quick-product')).toHaveLength(1)
    await wrapper.get('.quick-product').trigger('click')
    await button('港码 12').trigger('click')
    await button('加入购物袋').trigger('click')
    expect(wrapper.get('.s-bag-line').text()).toContain('港码 12')
    expect(store.state.bag.items[0].productId).toBe('p2')
  })

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

it('空商城没有固定分类，后台新建的空分类也可以选择', async () => {
  vi.useFakeTimers()
  const settings = makeCatalog().settings
  api.mockResolvedValueOnce({products:[], categories:[], settings})
  await render()
  expect(wrapper.find('.quick-sidebar').exists()).toBe(false)
  expect(wrapper.findAll('.quick-product')).toHaveLength(0)
  api.mockResolvedValueOnce({products:[], categories:[{id:11,name:'新分类',position:0}], settings})
  await vi.advanceTimersByTimeAsync(5000)
  await flushPromises()
  await button('新分类').trigger('click')
  expect(wrapper.get('.quick-listing h2').text()).toBe('新分类')
  api.mockResolvedValueOnce({products:[], categories:[{id:11,name:'已改名',position:0}], settings})
  await vi.advanceTimersByTimeAsync(5000)
  await flushPromises()
  expect(wrapper.get('.quick-listing h2').text()).toBe('已改名')
})

it('使用后台封面，购物袋和预约分两步', async () => {
  const catalog = makeCatalog()
  catalog.settings.quickCover = '/api/v1/media/fixture'
  api.mockResolvedValueOnce(catalog)
  await render()
  expect(wrapper.get('.quick-cover img').attributes('src')).toBe('/api/v1/media/fixture')
  await wrapper.get('button[aria-label="查看初光 · 经典莫桑钻戒"]').trigger('click')
  await button('港码 12').trigger('click')
  await wrapper.get('.detail-title button').trigger('click')
  expect(store.state.bag.items[0].productId).toBe('p2')
  await wrapper.get('button[aria-label="关闭窗口"]').trigger('click')
  await wrapper.get('button[aria-label="打开购物袋"]').trigger('click')
  expect(wrapper.find('.checkout-information').exists()).toBe(false)
  await button('继续填写预约').trigger('click')
  expect(wrapper.find('.checkout-information').exists()).toBe(true)
})
