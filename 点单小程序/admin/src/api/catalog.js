import request from '@/utils/request'
export const loadCatalog=()=>request.get('/admin/catalog')
export const saveProduct=(id,body)=>request.put(`/admin/products/${encodeURIComponent(id)}`,body)
export const saveSettings=body=>request.put('/admin/settings',body)
export const loadDashboard=(summary=false)=>request.get('/admin/dashboard',{params:summary?{summary:1}:undefined})
export const saveCategory=(id,body)=>request.put(`/admin/categories/${encodeURIComponent(id)}`,body)
export const deleteCategory=id=>request.delete(`/admin/categories/${encodeURIComponent(id)}`)
export const uploadImage=file=>{
  const data=new FormData()
  data.append('image',file)
  return request.post('/admin/media',data)
}
