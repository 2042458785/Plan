import {loadCatalog} from './catalog'
import {createReservation,getReservation} from './orders'
// 顾客端仅声明公开商品和预约接口，不打包任何管理 API。
export async function shopApi(path:string,options:any={}){
 if(path==='/catalog')return loadCatalog()
 if(path==='/orders'&&options.method==='POST')return createReservation(JSON.parse(options.body))
 if(path.startsWith('/orders/'))return getReservation(path.slice('/orders/'.length),options.headers?.['X-Order-Token']||'')
 throw new Error('接口不存在')
}
