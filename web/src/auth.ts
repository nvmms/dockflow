import { reactive } from 'vue'
import { api } from './api'

export const auth = reactive({ username: '', ready: false })

export async function refreshSession() {
  try {
    const value = await api.get<{ username: string }>('/auth/session')
    auth.username = value.username
    return true
  } catch {
    auth.username = ''
    return false
  } finally {
    auth.ready = true
  }
}
