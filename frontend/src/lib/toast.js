import { reactive } from 'vue'

const state = reactive({ msg: '', kind: 'ok' })
let timer = null

export function show(msg, kind = 'ok') {
  state.msg = msg
  state.kind = kind
  clearTimeout(timer)
  timer = setTimeout(() => (state.msg = ''), 3200)
}

export function useToast() {
  return state
}