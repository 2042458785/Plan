import axios from 'axios'
import { getBaseURL, APP_CONFIG } from '@/config'
import { logger } from '@/utils/logger'
const request = axios.create({ baseURL:getBaseURL(), timeout:APP_CONFIG.timeout })
request.interceptors.response.use(response=>{
  if(response.data?.code!==200) return Promise.reject(new Error(response.data?.msg || '请求失败'))
  return response
},error=>{
  logger.warn('request','response','接口请求失败')
  return Promise.reject(Object.assign(new Error(error.response?.data?.msg || '暂时无法连接精品店，请稍后重试'), {status:error.response?.status}))
})
export default request
export function randomKey():string {const b=new Uint8Array(24);window.crypto.getRandomValues(b);return Array.from(b,n=>n.toString(16).padStart(2,'0')).join('')}
