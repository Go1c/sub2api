import axios, { type AxiosInstance } from 'axios'

const baseURL = import.meta.env.VITE_API_BASE_URL || '/api/v1'

export const http: AxiosInstance = axios.create({
  baseURL,
  timeout: 15000
})

http.interceptors.request.use((config) => {
  const token = localStorage.getItem('dc_token')
  if (token) {
    config.headers = config.headers || {}
    config.headers['Authorization'] = `Bearer ${token}`
  }
  return config
})

http.interceptors.response.use(
  (res) => res,
  (err) => {
    if (err?.response?.status === 401) {
      localStorage.removeItem('dc_token')
      localStorage.removeItem('dc_user')
      if (location.pathname !== '/login') {
        location.replace('/login')
      }
    }
    return Promise.reject(err)
  }
)
