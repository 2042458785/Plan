import Vue from 'vue'
import VueRouter from 'vue-router'
import store from '@/store'
import {session} from '@/api/auth'
import Layout from '@/components/AdminLayout.vue'
Vue.use(VueRouter)
const router=new VueRouter({mode:'history',base:process.env.BASE_URL,routes:[
 {path:'/login',component:()=>import('@/views/LoginView.vue'),meta:{title:'商家登录'}},
 {path:'/',component:Layout,meta:{requireAuth:true},children:[{path:'',redirect:'orders'},{path:'orders',component:()=>import('@/views/OrdersView.vue'),meta:{title:'预约工作台',requireAuth:true}},{path:'products',component:()=>import('@/views/ProductsView.vue'),meta:{title:'商品与库存',requireAuth:true}},{path:'overview',component:()=>import('@/views/OverviewView.vue'),meta:{title:'营业概览',requireAuth:true}},{path:'settings',component:()=>import('@/views/SettingsView.vue'),meta:{title:'店铺设置',requireAuth:true}}]},
 {path:'*',redirect:'/orders'},
]})
router.beforeEach(async(to,from,next)=>{document.title=`素序 · ${to.meta.title||'商家工作室'}`;if(to.matched.some(r=>r.meta.requireAuth)){if(!store.getters['UBD/getUserToken']){next('/login');return}try{await session();next()}catch{next('/login')}}else next()})
window.addEventListener('suxu:unauthorized',()=>{if(router.currentRoute.path!=='/login')router.replace('/login').catch(()=>{})})
export default router
