import request from '@/utils/request'
export async function loadCatalog(){return (await request.get('/catalog')).data.data}
