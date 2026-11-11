import {createStore} from 'vuex'
function read(key:string,fallback:any){try{return JSON.parse(localStorage.getItem(key)||'null')??fallback}catch{return fallback}}
export default createStore({
 modules:{
  bag:{namespaced:true,state:()=>({items:read('suxu-bag-v1',[])}),mutations:{setItems(state:any,items:any[]){state.items=items;localStorage.setItem('suxu-bag-v1',JSON.stringify(items))}}},
  wishlist:{namespaced:true,state:()=>({ids:read('suxu-favorites-v1',[])}),mutations:{setIds(state:any,ids:string[]){state.ids=ids;localStorage.setItem('suxu-favorites-v1',JSON.stringify(ids))}}},
 },
})
