import request from '@/utils/request'
export const loadOrders=params=>request.get('/admin/orders',{params})
export const updateOrder=(id,body)=>request.patch(`/admin/orders/${encodeURIComponent(id)}`,body)
export const STATUS={pending:'待确认',confirmed:'备货中',ready:'待自提',completed:'已完成',cancelled:'已取消'}
