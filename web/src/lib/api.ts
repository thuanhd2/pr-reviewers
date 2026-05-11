const BASE = '/api'

interface ApiResponse<T> {
  code: number
  data: T
  error?: string
}

export interface ListData<T> {
  items: T[]
  meta: { page: number; per_page: number; total: number }
}

async function request<T>(url: string, options?: RequestInit): Promise<T> {
  const res = await fetch(`${BASE}${url}`, {
    headers: { 'Content-Type': 'application/json' },
    ...options,
  })
  const json: ApiResponse<T> = await res.json()
  if (json.code !== 0) throw new Error(json.error || 'API error')
  return json.data
}

export const api = {
  getPRs: (params: Record<string, string> = {}) => {
    const qs = new URLSearchParams(params).toString()
    return request<ListData<any>>(`/prs?${qs}`)
  },
  getPR: (id: number) => request<any>(`/prs/${id}`),
  refreshPR: (id: number) => request<any>(`/prs/${id}/refresh`, { method: 'POST' }),
  getReviews: (prId: number) => request<any[]>(`/prs/${prId}/reviews`),
  getReview: (id: number) => request<any>(`/reviews/${id}`),
  updateReview: (id: number, data: any) =>
    request<any>(`/reviews/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
  approveReview: (id: number) =>
    request<any>(`/reviews/${id}/approve`, { method: 'POST' }),
  rejectReview: (id: number) =>
    request<any>(`/reviews/${id}/reject`, { method: 'POST' }),
  rerunReview: (id: number) =>
    request<any>(`/reviews/${id}/rerun`, { method: 'POST' }),
  getRepoConfigs: () => request<any[]>('/configs/repos'),
  createRepoConfig: (data: any) =>
    request<any>('/configs/repos', { method: 'POST', body: JSON.stringify(data) }),
  updateRepoConfig: (id: number, data: any) =>
    request<any>(`/configs/repos/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
  deleteRepoConfig: (id: number) =>
    request<any>(`/configs/repos/${id}`, { method: 'DELETE' }),
  testRepoConfig: (id: number) =>
    request<any>(`/configs/repos/${id}/test`, { method: 'POST' }),
  getCLIConfigs: () => request<string[]>('/configs/clis'),
  getDashboard: () => request<any>('/dashboard'),
  getHistory: (params: Record<string, string> = {}) => {
    const qs = new URLSearchParams(params).toString()
    return request<ListData<any>>(`/history?${qs}`)
  },
  getSchedulerJobs: () => request<any[]>('/scheduler/jobs'),
}
