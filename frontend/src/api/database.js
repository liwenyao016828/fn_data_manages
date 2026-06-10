import axios from 'axios'

const baseURL = '/api/databases'

export const databaseApi = {
  create: (data) => axios.post(`${baseURL}/db`, data),
  search: (data) => axios.post(`${baseURL}/db/search`, data),
  get: (id) => axios.get(`${baseURL}/db/${id}`),
  revealPassword: (id) => axios.get(`${baseURL}/db/${id}?reveal=true`),
  update: (data) => axios.put(`${baseURL}/db`, data),
  delete: (data) => axios.delete(`${baseURL}/db`, { data }),
  list: (type) => axios.get(`${baseURL}/db/list/${type}`),
  check: (data) => axios.post(`${baseURL}/db/check`, data)
}

export const detectApi = {
  list: (includeIgnored = false) =>
    axios.get(`${baseURL}/detect${includeIgnored ? '?all=true' : ''}`),
  scan: () => axios.post(`${baseURL}/detect`),
  ignoredList: () => axios.get(`${baseURL}/detect/ignore`),
  ignore: (fingerprint, label = '') =>
    axios.post(`${baseURL}/detect/ignore`, { fingerprint, label }),
  unignore: (fingerprint) =>
    axios.delete(`${baseURL}/detect/ignore`, { params: { fingerprint } })
}