// 管理端独立配置，不从顾客工程导入代码或令牌。
export const APP_CONFIG = {
  apiBase: '/api/v1',
  timeout: 15000,
  // 生产环境统一使用80端口，商城在根路径，管理端在/admin/。
  storefrontURL: process.env.NODE_ENV === 'production'
    ? `${window.location.origin}/`
    : `${window.location.protocol}//${window.location.hostname}:8081/`,
  quickStorefrontURL: process.env.NODE_ENV === 'production'
    ? `${window.location.origin}/quick/`
    : `${window.location.protocol}//${window.location.hostname}:8083/quick/`,
}
