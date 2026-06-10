import axios from 'axios'

const baseURL = '/api/health'

export const healthApi = {
  getAll: () => axios.get(`${baseURL}/check`),
  getOne: (uid) => axios.get(`${baseURL}/check`, { params: { uid } }),
  forceCheck: (uid) => axios.post(`${baseURL}/check`, { uid }),
  forceCheckAll: () => axios.post(`${baseURL}/check`, {}),
  forceCheckByUID: (uid) => axios.post(`${baseURL}/check/${uid}`),

  getConfig: () => axios.get(`${baseURL}/config`),
  updateConfig: (config) => axios.put(`${baseURL}/config`, config),
}
