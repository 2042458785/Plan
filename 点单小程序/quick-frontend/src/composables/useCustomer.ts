import { computed, ref } from 'vue'
import { shopApi } from '@/api'
import type { CustomerAccount } from '@/types/catalog'

const storageKey = 'suxu-customer-session-v1'

export function useCustomer(notify: (message: string) => void) {
  const secure = location.protocol === 'https:' || ['localhost', '127.0.0.1', '[::1]'].includes(location.hostname)
  const token = ref('')
  const account = ref<CustomerAccount | null>(null)
  const loading = ref(false)
  const authError = ref('')
  const signedIn = computed(() => Boolean(account.value && token.value))

  function remember(value: string) {
    token.value = value
    try {
      if (value) localStorage.setItem(storageKey, value)
      else localStorage.removeItem(storageKey)
    } catch {
      if (value) notify('本次已登录，浏览器无法记住会话')
    }
  }

  function clear() {
    remember('')
    account.value = null
  }

  async function request(path: string, method = 'GET', data?: unknown) {
    const activeToken = token.value
    try {
      return await shopApi('/customers' + path, {
        method, body: data === undefined ? undefined : JSON.stringify(data),
        headers: activeToken ? { Authorization: 'Bearer ' + activeToken } : {},
      })
    } catch (error) {
      if (error.status === 401 && activeToken && token.value === activeToken) clear()
      throw error
    }
  }

  async function restore() {
    if (!secure || loading.value || account.value) return
    try { token.value = localStorage.getItem(storageKey) || '' } catch { return }
    if (!token.value) return
    loading.value = true
    try { account.value = await request('/profile') }
    catch (error) { if (error.status !== 401) authError.value = error.message }
    finally { loading.value = false }
  }

  async function authenticate(register: boolean, input: { name: string; contact: string; password: string }) {
    if (!secure || loading.value) return false
    loading.value = true
    authError.value = ''
    try {
      const session = await request(register ? '/register' : '/login', 'POST', input)
      remember(session.token)
      account.value = session.account
      return true
    } catch (error) {
      authError.value = error.message
      return false
    } finally { loading.value = false }
  }

  async function save(input: { name: string; dorm: string }) {
    loading.value = true
    authError.value = ''
    try {
      account.value = await request('/profile', 'PUT', input)
      notify('顾客资料已保存')
    } catch (error) { authError.value = error.message }
    finally { loading.value = false }
  }

  async function logout() {
    loading.value = true
    authError.value = ''
    try { await request('/logout', 'POST', {}); clear() }
    catch (error) { if (error.status !== 401) authError.value = error.message }
    finally { loading.value = false }
  }

  return { secure, account, loading, authError, signedIn, request, restore, authenticate, save, logout }
}
