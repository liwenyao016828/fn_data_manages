import axios from 'axios'

const baseURL = '/api/databases'

export const databaseApi = {
  create: (data) => axios.post(`${baseURL}/db`, data),
  search: (data) => axios.post(`${baseURL}/db/search`, data),
  get: (id) => axios.get(`${baseURL}/db/${id}`),
  update: (data) => axios.put(`${baseURL}/db`, data),
  delete: (data) => axios.delete(`${baseURL}/db`, { data }),
  list: (type) => axios.get(`${baseURL}/db/list/${type}`),
  check: (data) => axios.post(`${baseURL}/db/check`, data)
}