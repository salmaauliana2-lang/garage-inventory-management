import axios from 'axios'
const api = axios.create({ baseURL: import.meta.env.VITE_API_URL || 'http://localhost:8080/api/v1' })
api.interceptors.request.use(config => {
  const token = localStorage.getItem('garage_token')
  if (token) config.headers.Authorization = `Bearer ${token}`
  return config
})
api.interceptors.response.use(r=>r, err=>{
  if (err.response?.status===401) { localStorage.removeItem('garage_token'); localStorage.removeItem('garage_user') }
  return Promise.reject(err)
})
export default api
