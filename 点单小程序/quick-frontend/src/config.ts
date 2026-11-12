/** 与参考顾客端一致：APP_CONFIG 是接口与静态资源的唯一配置源。 */
export const APP_CONFIG = {
  http_domain: '', // IP 部署保持留空，由当前站点的 Nginx 代理 API。
  http_protocol: 'http',
  api_prefix: '/api/v1',
  static_url: '',
  timeout: 15000,
}
export function getBaseURL(path = ''): string {
  return `${APP_CONFIG.http_domain ? `${APP_CONFIG.http_protocol}://${APP_CONFIG.http_domain}` : ''}${APP_CONFIG.api_prefix}${path}`
}
export function assetUrl(path: string): string {
  if (!path.startsWith('/images/')) return path
  return APP_CONFIG.static_url
    ? `${APP_CONFIG.static_url}${path}`
    : `${import.meta.env.BASE_URL}${path.slice(1)}`
}
