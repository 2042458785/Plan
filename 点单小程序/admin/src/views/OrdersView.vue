<template><div><div class="page-heading"><div><h1>预约工作台</h1><p>查看预约、收款和配送地址。</p></div><div class="heading-actions"><ShopQR/><el-button icon="el-icon-download" :loading="exporting" @click="exportCSV">导出订单</el-button></div></div><div class="metric-grid"><article><span>今日实收</span><strong>¥ {{ money(metrics.revenue||0) }}</strong><p>已收款且未取消的今日订单</p></article><article><span>今日预约</span><strong>{{ metrics.todayOrders||0 }}<small>单</small></strong></article><article class="metric-warm"><span>等待确认</span><strong>{{ metrics.pending||0 }}<small>单</small></strong></article><article><span>正在备货</span><strong>{{ metrics.confirmed||0 }}<small>单</small></strong></article></div><div v-if="connectionError" class="connection-alert" role="alert">{{ connectionError }}</div><section class="atelier-panel"><div class="panel-toolbar order-toolbar"><el-radio-group aria-label="筛选预约状态" v-model="status" size="small" @change="changeFilter"><el-radio-button v-for="item in filters" :key="item.value" :label="item.value">{{ item.label }}</el-radio-button></el-radio-group><el-input v-model="query" prefix-icon="el-icon-search" aria-label="搜索预约" placeholder="搜索预约号、顾客或商品" clearable @input="queueSearch"></el-input></div><div class="list-summary"><span>{{ total }} 笔预约 <small>按最新下单排序</small></span><span><i class="live-dot"></i>自动刷新</span></div><div v-if="!orders.length&&!loading" class="panel-empty"><i class="el-icon-document-copy"></i><h3>{{ query ? '没有匹配的预约' : '当前没有预约' }}</h3></div><div v-loading="loading" class="order-grid"><article v-for="o in orders" :key="o.id" class="reservation-card"><header><div><span>预约编号</span><h2>{{ o.number }}</h2></div><el-tag size="small" :type="tagType(o.status)" effect="plain">{{ orderStatus(o) }}</el-tag></header><div class="reservation-meta"><span>{{ o.customer||'到店顾客' }}<small v-if="o.customerNumber"> · {{ o.customerNumber }}</small></span><span>{{ time(o.createdAt) }}</span></div><div class="reservation-items"><div v-for="(item,index) in o.items" :key="index"><span>{{ item.name }} <b>×{{ item.quantity }}</b><small>{{ item.variant }} · {{ item.color }}</small></span><strong>¥ {{ money(item.price*item.quantity) }}</strong></div></div><div class="reservation-note">
  <strong>{{ o.fulfillment==='delivery'?'送货上门':'自提' }}</strong>
  <div v-if="o.contact">联系方式：{{ o.contact }}</div>
  <div v-if="o.fulfillment==='delivery'">配送地址：{{ o.address }}<br>配送费：¥ {{ money(o.deliveryFee) }}</div>
  <div v-else>{{ o.pickup||'营业时间内自提' }}</div>
</div><div v-if="o.note" class="reservation-note">留言：{{ o.note }}</div><div class="reservation-amount"><span>合计 <strong>¥ {{ money(o.total) }}</strong></span><el-button size="mini" :type="o.paid?'success':''" plain :loading="working===o.id" @click="patch(o,{paid:!o.paid})">{{ o.paid?'已收款':'标记收款' }}</el-button></div><div class="reservation-actions"><el-button v-if="next[o.status]" size="small" type="primary" :loading="working===o.id" @click="patch(o,{status:next[o.status]})">{{ o.status==='ready'&&o.fulfillment==='delivery'?'确认已送达':actions[o.status] }} <i class="el-icon-right"></i></el-button><el-button v-if="['pending','confirmed','ready'].includes(o.status)" size="small" type="text" @click="cancel(o)">取消预约</el-button><span v-if="!next[o.status]">{{ orderStatus(o) }}</span></div></article></div><div class="pagination-row"><el-pagination background layout="prev,pager,next,total" :current-page="page" :page-size="size" :total="total" @current-change="changePage"></el-pagination></div></section></div></template>
<script>
import ShopQR from '@/components/ShopQR.vue'
import {loadOrders,updateOrder,STATUS} from '@/api/orders'
import {loadDashboard} from '@/api/catalog'
import {money} from '@/utils/money'
export default{components:{ShopQR},data:()=>({orders:[],metrics:{},total:0,page:1,size:30,status:'active',query:'',loading:true,exporting:false,working:'',connectionError:'',timer:null,searchTimer:null,loadSequence:0,listPending:false,metricsPending:false,next:{pending:'confirmed',confirmed:'ready',ready:'completed'},actions:{pending:'确认并备货',confirmed:'备货完成',ready:'确认已自提'},filters:[{value:'active',label:'进行中'},{value:'pending',label:'待确认'},{value:'confirmed',label:'备货中'},{value:'ready',label:'待自提 / 配送'},{value:'completed',label:'已完成'},{value:'cancelled',label:'已取消'},{value:'all',label:'全部'}]}),mounted() {
  this.load()
  this.timer = setInterval(this.refreshVisible, 4000)
  document.addEventListener('visibilitychange', this.refreshVisible)
},beforeDestroy() {
  clearInterval(this.timer)
  clearTimeout(this.searchTimer)
  document.removeEventListener('visibilitychange', this.refreshVisible)
  this.loadSequence++
},methods:{money,orderStatus(o){return o.status==='ready'&&o.fulfillment==='delivery'?'待配送':STATUS[o.status]},time(d){return new Date(d).toLocaleString('zh-CN',{month:'2-digit',day:'2-digit',hour:'2-digit',minute:'2-digit'})},tagType(s){return {pending:'warning',confirmed:'',ready:'success',completed:'info',cancelled:'info'}[s]},refreshVisible() {
  if (!document.hidden && !this.listPending && !this.working && !this.searchTimer) this.load(false)
},
async refreshMetrics() {
  if (this.metricsPending) return
  this.metricsPending = true
  try {
    this.metrics = await loadDashboard(true)
  } catch {
    // 列表独立更新，统计请求失败时保留上一次结果。
  } finally {
    this.metricsPending = false
  }
},
async load(show = true) {
  const sequence = ++this.loadSequence
  if (show) this.loading = true
  this.listPending = true
  void this.refreshMetrics()
  try {
    const data = await loadOrders({ page: this.page, size: this.size, status: this.status, search: this.query.trim() })
    if (sequence !== this.loadSequence) return
    const lastPage = Math.max(1, Math.ceil(data.total / this.size))
    if (this.page > lastPage) {
      this.page = lastPage
      await this.load(show)
      return
    }
    this.orders = data.items || []
    this.total = data.total
    this.connectionError = ''
  } catch (error) {
    if (sequence === this.loadSequence) this.connectionError = error.message
  } finally {
    if (sequence === this.loadSequence) {
      this.loading = false
      this.listPending = false
    }
  }
},
changeFilter() {
  clearTimeout(this.searchTimer)
  this.searchTimer = null
  this.page = 1
  this.load()
},
queueSearch() {
  clearTimeout(this.searchTimer)
  this.loadSequence++
  this.searchTimer = setTimeout(this.changeFilter, 300)
},
changePage(page) {
  this.page = page
  this.load()
},
async patch(o,body){if(this.working)return;this.working=o.id;try{await updateOrder(o.id,{...body,version:o.version});await this.load(false);this.$message.success('预约已更新')}catch(e){this.$message.error(e.message);await this.load(false)}finally{this.working=''}},async cancel(o){try{await this.$confirm('取消后返还仍在目录中的商品库存。若已收款，请先与顾客确认线下退款安排。','取消预约',{confirmButtonText:'确认取消',cancelButtonText:'保留预约',type:'warning'});await this.patch(o,{status:'cancelled'})}catch{}},async exportCSV(){this.exporting=true;try{let rows=[],page=1,total=1;while(rows.length<total){const data=await loadOrders({page,size:200,status:'all'});total=data.total;if(!data.items.length)break;rows.push(...data.items);page++}const data=[['预约号','顾客','顾客编号','联系方式','取货方式','地址 / 自提安排','商品与规格','配送费（元）','金额（元）','状态','收款'],...rows.map(o=>[o.number,o.customer,o.customerNumber,o.contact,o.fulfillment==='delivery'?'送货上门':'自提',o.address||o.pickup,o.items.map(i=>`${i.name} ${i.variant} ${i.color} ×${i.quantity}`).join(' / '),money(o.deliveryFee||0),money(o.total),this.orderStatus(o),o.paid?'已收款':'未收款'])];const escape=v=>'"'+String(v==null?'':v).replace(/^[=+@\-\t\r]/,"'$&").replace(/"/g,'""')+'"';const url=URL.createObjectURL(new Blob(['\ufeff'+data.map(r=>r.map(escape).join(',')).join('\r\n')],{type:'text/csv;charset=utf-8;'}));const a=document.createElement('a');a.href=url;a.download='素序预约订单.csv';a.click();URL.revokeObjectURL(url)}catch(e){this.$message.error(e.message)}finally{this.exporting=false}}}}
</script>
