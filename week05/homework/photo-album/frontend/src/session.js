import { shallowRef } from 'vue'

const key = 'photo-album-session'

function readSession() {
  try {
    const value = JSON.parse(localStorage.getItem(key))
    return typeof value?.token === 'string' && value.user?.id ? value : null
  } catch {
    return null
  }
}

export const session = shallowRef(readSession())

export function setSession(value) {
  if (value) {
    localStorage.setItem(key, JSON.stringify(value))
  } else {
    localStorage.removeItem(key)
  }
  session.value = value
}

window.addEventListener('storage', (event) => {
  if (event.key === key || event.key === null) session.value = readSession()
})
