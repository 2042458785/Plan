import request from '@/utils/request'
export const login=body=>request.post('/admin/login',body)
export const session=()=>request.get('/admin/session')
export const logout=()=>request.post('/admin/logout',{})
