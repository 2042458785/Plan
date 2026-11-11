import request from '@/utils/request'
export async function createReservation(body:unknown){return (await request.post('/orders',body)).data.data}
export async function getReservation(id:string,token:string){return (await request.get(`/orders/${encodeURIComponent(id)}`,{headers:{'X-Order-Token':token}})).data.data}
