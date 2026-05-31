export function sourceParam(isRemote) {
  return isRemote ? 'source=remote' : 'source=local'
}

export function sourceValue(isRemote) {
  return isRemote ? 'remote' : 'local'
}

export function instanceUid(inst) {
  return (inst.isRemote ? 'r:' : 'l:') + inst.id
}
