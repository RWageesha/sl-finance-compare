// Shared admin session state — one /me fetch backs every page's "who am
// I, what role do I have" instead of each page re-fetching it. The
// underlying session itself is an HttpOnly cookie the browser manages;
// this just mirrors what the server says about it.
export interface AdminMe {
  id: number
  username: string
  email: string
  role: 'super_admin' | 'admin' | 'editor' | 'viewer'
}

export function useAdminAuth() {
  const admin = useState<AdminMe | null>('admin-user', () => null)

  async function refresh(): Promise<boolean> {
    try {
      admin.value = await $fetch<AdminMe>('/api/v1/admin/me', { credentials: 'include' })
      return true
    } catch {
      admin.value = null
      return false
    }
  }

  async function logout() {
    await $fetch('/api/v1/admin/logout', { method: 'POST', credentials: 'include' }).catch(() => {})
    admin.value = null
    await navigateTo('/admin/login')
  }

  const ROLE_RANK: Record<string, number> = { viewer: 0, editor: 1, admin: 2, super_admin: 3 }
  function roleAtLeast(min: string): boolean {
    if (!admin.value) return false
    return (ROLE_RANK[admin.value.role] ?? -1) >= (ROLE_RANK[min] ?? 99)
  }

  return { admin, refresh, logout, roleAtLeast }
}
