import Vue from 'vue'
import Vuex from 'vuex'
Vue.use(Vuex)
let user=null
try{user=JSON.parse(localStorage.getItem('suxu-merchant-user')||'null')}catch{user=null}
export default new Vuex.Store({modules:{UBD:{namespaced:true,state:{token:localStorage.getItem('suxu-merchant-token')||'',user},getters:{getUserToken:s=>s.token},mutations:{setSession(s,payload){s.token=payload.token;s.user=payload.user;localStorage.setItem('suxu-merchant-token',s.token);localStorage.setItem('suxu-merchant-user',JSON.stringify(s.user))},clearAuthSession(s){s.token='';s.user=null;localStorage.removeItem('suxu-merchant-token');localStorage.removeItem('suxu-merchant-user')}}}}})
