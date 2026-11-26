import { session, setSession } from './session'

const messages = {
  400: '请检查填写的信息或选择的文件。',
  401: '请登录后继续。',
  403: '你没有权限访问此相册或执行此操作。',
  404: '找不到这个相册或资源。',
  409: '这个用户名已被使用。',
  413: '图片大小不能超过 5 MiB。',
  415: '请选择有效的 JPEG、PNG 或 WebP 图片。',
}

export async function api(path, { method = 'GET', body, signal, auth = true } = {}) {
  const token = auth ? session.value?.token : undefined
  const headers = new Headers()
  if (token) headers.set('Authorization', `Bearer ${token}`)
  if (body && !(body instanceof FormData)) {
    headers.set('Content-Type', 'application/json')
    body = JSON.stringify(body)
  }

  let response
  try {
    response = await fetch(`/api/v1${path}`, { method, body, headers, signal })
  } catch (error) {
    if (error.name === 'AbortError') throw error
    throw new Error('连接服务失败，请检查后端后重试。')
  }
  if (!response.ok) {
    const expired = response.status === 401 && token && session.value?.token === token
    if (expired) setSession(null)
    throw Object.assign(new Error(expired ? '登录已失效，请重新登录。' : messages[response.status] ?? '服务暂时无法处理请求，请重试。'), {
      status: response.status,
    })
  }
  const result = await response.json()
  if (result.code !== 0) throw new Error('服务返回了无法处理的结果。')
  return result.data
}
