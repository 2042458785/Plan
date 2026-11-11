import axios from 'axios'
import store from '@/store'
import {APP_CONFIG} from '@/config'
import {logger} from './logger'
const request=axios.create({baseURL:APP_CONFIG.apiBase,timeout:APP_CONFIG.timeout})
request.interceptors.request.use(config=>{const token=store.getters['UBD/getUserToken'];if(token)config.headers.Authorization=`Bearer ${token}`;return config})
request.interceptors.response.use(response=>{if(response.data.code!==200)return Promise.reject(new Error(response.data.msg||'请求失败'));return response.data.data},error=>{if(error.response&&error.response.status===401){store.commit('UBD/clearAuthSession');window.dispatchEvent(new Event('suxu:unauthorized'))}logger.warn('request','response','管理接口请求失败');const e=new Error(error.response&&error.response.data&&error.response.data.msg||'无法连接后端，请检查 API 服务');e.status=error.response&&error.response.status;return Promise.reject(e)})
export default request
