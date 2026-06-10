import axios from 'axios'

const baseURL = '/api/health'

const makeSignal = () => new AbortController()

const isAbort = (e) =>
  !!e &&
  (axios.isCancel?.(e) ||
    e.code === 'ERR_CANCELED' ||
    e.name === 'CanceledError' ||
    e.name === 'AbortError' ||
    (typeof e.message === 'string' && /cancel|abort/i.test(e.message)))

export const healthApi = {
  getAll: (signal) => axios.get(`${baseURL}/check`, { signal }),
  getOne: (uid, signal) => axios.get(`${baseURL}/check`, { params: { uid }, signal }),
  forceCheck: (uid, signal) => axios.post(`${baseURL}/check`, { uid }, { signal }),
  forceCheckAll: (signal) => axios.post(`${baseURL}/check`, {}, { signal }),
  forceCheckByUID: (uid, signal) => axios.post(`${baseURL}/check/${uid}`, {}, { signal }),

  getConfig: (signal) => axios.get(`${baseURL}/config`, { signal }),
  updateConfig: (config, signal) => axios.put(`${baseURL}/config`, config, { signal }),

  makeSignal,
  isAbort,
}
