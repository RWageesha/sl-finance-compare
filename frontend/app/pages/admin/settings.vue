<script setup lang="ts">
useHead({ title: 'Settings — FindRate LK', meta: [{ name: 'robots', content: 'noindex, nofollow' }] })

const { admin, roleAtLeast, refresh } = useAdminAuth()

const newUsername = ref(admin.value?.username ?? '')
const usernameError = ref('')
const usernameToast = ref('')
async function saveUsername() {
  usernameError.value = ''
  try {
    await $fetch('/api/v1/admin/me', { method: 'PATCH', credentials: 'include', body: { username: newUsername.value } })
    await refresh()
    usernameToast.value = 'Username updated.'
  } catch (e: any) {
    usernameError.value = e?.data?.error || 'Update failed'
  }
}

const currentPassword = ref('')
const newPassword = ref('')
const confirmPassword = ref('')
const passwordError = ref('')
const passwordToast = ref('')
async function savePassword() {
  passwordError.value = ''
  if (newPassword.value !== confirmPassword.value) {
    passwordError.value = "Passwords don't match"
    return
  }
  try {
    await $fetch('/api/v1/admin/me/password', { method: 'PATCH', credentials: 'include', body: { current_password: currentPassword.value, new_password: newPassword.value } })
    currentPassword.value = ''; newPassword.value = ''; confirmPassword.value = ''
    passwordToast.value = 'Password updated.'
  } catch (e: any) {
    passwordError.value = e?.data?.error || 'Update failed'
  }
}

// ===== Team Members (super_admin only) =====
interface Member { id: number; username: string; email: string; role: string; status: string; last_login_at: string | null }
const members = ref<Member[]>([])
const teamLoading = ref(false)
const teamToast = ref('')

async function loadTeam() {
  if (!roleAtLeast('super_admin')) return
  teamLoading.value = true
  try {
    members.value = (await $fetch<{ data: Member[] }>('/api/v1/admin/team', { credentials: 'include' })).data ?? []
  } finally {
    teamLoading.value = false
  }
}
onMounted(loadTeam)

const addingMember = ref(false)
const addForm = reactive({ username: '', email: '', password: '', role: 'editor' })
const addError = ref('')
async function addMember() {
  addError.value = ''
  try {
    await $fetch('/api/v1/admin/team', { method: 'POST', credentials: 'include', body: addForm })
    addingMember.value = false
    addForm.username = ''; addForm.email = ''; addForm.password = ''; addForm.role = 'editor'
    teamToast.value = 'Team member added.'
    await loadTeam()
  } catch (e: any) {
    addError.value = e?.data?.error || 'Failed to add member'
  }
}

const editingMember = ref<Member | null>(null)
const editForm = reactive({ email: '', role: '' })
function openEditMember(m: Member) {
  editingMember.value = m
  editForm.email = m.email
  editForm.role = m.role
}
async function saveEditMember() {
  if (!editingMember.value) return
  try {
    await $fetch(`/api/v1/admin/team/${editingMember.value.id}`, { method: 'PATCH', credentials: 'include', body: editForm })
    editingMember.value = null
    teamToast.value = 'Member updated.'
    await loadTeam()
  } catch {
    teamToast.value = ''
  }
}

const confirmTarget = ref<{ member: Member; action: 'disable' | 'enable' } | null>(null)
async function toggleStatus() {
  if (!confirmTarget.value) return
  const { member, action } = confirmTarget.value
  try {
    await $fetch(`/api/v1/admin/team/${member.id}/${action}`, { method: 'PATCH', credentials: 'include' })
    teamToast.value = `Member ${action}d.`
  } catch {
    teamToast.value = ''
  } finally {
    confirmTarget.value = null
    await loadTeam()
  }
}

const resettingMember = ref<Member | null>(null)
const resetPassword = ref('')
async function confirmResetPassword() {
  if (!resettingMember.value) return
  try {
    await $fetch(`/api/v1/admin/team/${resettingMember.value.id}/reset-password`, { method: 'POST', credentials: 'include', body: { new_password: resetPassword.value } })
    teamToast.value = 'Password reset.'
  } catch {
    teamToast.value = ''
  } finally {
    resettingMember.value = null
    resetPassword.value = ''
  }
}
</script>

<template>
  <AdminShell>
    <h1 class="mb-5 text-xl font-bold text-navy">Settings</h1>

    <section class="mb-6 rounded-card border border-card-border bg-card p-5">
      <h2 class="mb-4 text-base font-bold text-navy">My Account</h2>

      <div class="mb-5 max-w-sm">
        <label class="mb-1 block text-sm font-semibold text-navy">Username</label>
        <div class="flex gap-2">
          <input v-model="newUsername" type="text" class="flex-1 rounded-lg border border-card-border px-3 py-2 text-sm">
          <button type="button" class="rounded-lg bg-primary px-4 py-2 text-sm font-bold text-white hover:bg-primary/90" @click="saveUsername">Save</button>
        </div>
        <p v-if="usernameError" class="mt-1 text-xs text-red-600">{{ usernameError }}</p>
      </div>

      <div class="max-w-sm">
        <label class="mb-1 block text-sm font-semibold text-navy">Change Password</label>
        <input v-model="currentPassword" type="password" placeholder="Current password" class="mb-2 w-full rounded-lg border border-card-border px-3 py-2 text-sm">
        <input v-model="newPassword" type="password" placeholder="New password" class="mb-2 w-full rounded-lg border border-card-border px-3 py-2 text-sm">
        <input v-model="confirmPassword" type="password" placeholder="Confirm new password" class="mb-2 w-full rounded-lg border border-card-border px-3 py-2 text-sm">
        <p v-if="passwordError" class="mb-2 text-xs text-red-600">{{ passwordError }}</p>
        <button type="button" class="rounded-lg bg-primary px-4 py-2 text-sm font-bold text-white hover:bg-primary/90" @click="savePassword">Update Password</button>
      </div>
    </section>

    <section v-if="roleAtLeast('super_admin')" class="rounded-card border border-card-border bg-card p-5">
      <div class="mb-4 flex items-center justify-between">
        <h2 class="text-base font-bold text-navy">Team Members</h2>
        <button type="button" class="rounded-lg bg-primary px-4 py-2 text-sm font-bold text-white hover:bg-primary/90" @click="addingMember = true">+ Add Team Member</button>
      </div>

      <LoadingState v-if="teamLoading" />
      <div v-else class="overflow-x-auto">
        <table class="w-full min-w-[640px] text-sm">
          <thead>
            <tr class="border-b border-card-border text-left text-[11px] uppercase tracking-wide text-muted">
              <th class="py-2 font-semibold">Username</th>
              <th class="py-2 font-semibold">Email</th>
              <th class="py-2 font-semibold">Role</th>
              <th class="py-2 font-semibold">Status</th>
              <th class="py-2 font-semibold">Last Login</th>
              <th class="py-2 font-semibold">Actions</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="m in members" :key="m.id" class="border-b border-card-border last:border-none">
              <td class="py-2 font-semibold text-navy">{{ m.username }}</td>
              <td class="py-2 text-muted">{{ m.email }}</td>
              <td class="py-2 text-muted">{{ m.role }}</td>
              <td class="py-2"><span class="rounded-pill px-2 py-0.5 text-[11px] font-bold" :class="m.status === 'active' ? 'bg-emerald-50 text-emerald-700' : 'bg-page text-muted'">{{ m.status }}</span></td>
              <td class="py-2 text-muted">{{ m.last_login_at ? fmtRelativeDate(m.last_login_at) : 'Never' }}</td>
              <td class="py-2">
                <div class="flex flex-wrap gap-2 text-xs font-bold">
                  <button v-if="m.role !== 'super_admin'" type="button" class="text-primary hover:underline" @click="openEditMember(m)">Edit</button>
                  <button v-if="m.role !== 'super_admin'" type="button" class="text-primary hover:underline" @click="confirmTarget = { member: m, action: m.status === 'active' ? 'disable' : 'enable' }">
                    {{ m.status === 'active' ? 'Disable' : 'Enable' }}
                  </button>
                  <button v-if="m.role !== 'super_admin'" type="button" class="text-primary hover:underline" @click="resettingMember = m">Reset Password</button>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>

    <div v-if="addingMember" class="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4">
      <div class="w-full max-w-sm rounded-card bg-card p-6">
        <h3 class="text-base font-bold text-navy">Add Team Member</h3>
        <div class="mt-4 flex flex-col gap-3">
          <input v-model="addForm.username" type="text" placeholder="Username" class="rounded-lg border border-card-border px-3 py-2 text-sm">
          <input v-model="addForm.email" type="email" placeholder="Email" class="rounded-lg border border-card-border px-3 py-2 text-sm">
          <input v-model="addForm.password" type="password" placeholder="Temporary password" class="rounded-lg border border-card-border px-3 py-2 text-sm">
          <select v-model="addForm.role" class="rounded-lg border border-card-border px-3 py-2 text-sm">
            <option value="admin">Admin</option>
            <option value="editor">Editor</option>
            <option value="viewer">Viewer</option>
          </select>
          <p v-if="addError" class="text-xs text-red-600">{{ addError }}</p>
        </div>
        <div class="mt-4 flex justify-end gap-2">
          <button type="button" class="rounded-lg border border-card-border px-4 py-2 text-sm font-semibold" @click="addingMember = false">Cancel</button>
          <button type="button" class="rounded-lg bg-primary px-4 py-2 text-sm font-bold text-white hover:bg-primary/90" @click="addMember">Add</button>
        </div>
      </div>
    </div>

    <div v-if="editingMember" class="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4">
      <div class="w-full max-w-sm rounded-card bg-card p-6">
        <h3 class="text-base font-bold text-navy">Edit {{ editingMember.username }}</h3>
        <div class="mt-4 flex flex-col gap-3">
          <input v-model="editForm.email" type="email" class="rounded-lg border border-card-border px-3 py-2 text-sm">
          <select v-model="editForm.role" class="rounded-lg border border-card-border px-3 py-2 text-sm">
            <option value="admin">Admin</option>
            <option value="editor">Editor</option>
            <option value="viewer">Viewer</option>
          </select>
        </div>
        <div class="mt-4 flex justify-end gap-2">
          <button type="button" class="rounded-lg border border-card-border px-4 py-2 text-sm font-semibold" @click="editingMember = null">Cancel</button>
          <button type="button" class="rounded-lg bg-primary px-4 py-2 text-sm font-bold text-white hover:bg-primary/90" @click="saveEditMember">Save</button>
        </div>
      </div>
    </div>

    <div v-if="resettingMember" class="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4">
      <div class="w-full max-w-sm rounded-card bg-card p-6">
        <h3 class="text-base font-bold text-navy">Reset password for {{ resettingMember.username }}</h3>
        <input v-model="resetPassword" type="password" placeholder="New temporary password" class="mt-4 w-full rounded-lg border border-card-border px-3 py-2 text-sm">
        <div class="mt-4 flex justify-end gap-2">
          <button type="button" class="rounded-lg border border-card-border px-4 py-2 text-sm font-semibold" @click="resettingMember = null">Cancel</button>
          <button type="button" class="rounded-lg bg-red-600 px-4 py-2 text-sm font-bold text-white hover:bg-red-700" @click="confirmResetPassword">Reset</button>
        </div>
      </div>
    </div>

    <ConfirmationModal
      v-if="confirmTarget"
      :message="`This will ${confirmTarget.action} ${confirmTarget.member.username}'s account.`"
      :confirm-label="confirmTarget.action === 'disable' ? 'Disable' : 'Enable'"
      @confirm="toggleStatus"
      @cancel="confirmTarget = null"
    />
    <SuccessToast v-if="usernameToast" :message="usernameToast" @dismiss="usernameToast = ''" />
    <SuccessToast v-if="passwordToast" :message="passwordToast" @dismiss="passwordToast = ''" />
    <SuccessToast v-if="teamToast" :message="teamToast" @dismiss="teamToast = ''" />
  </AdminShell>
</template>
