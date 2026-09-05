// Copies the standalone ES-module Vue runtime into the build output so the
// console shell can serve it at /admin/mf/vue.js via <script type="importmap">.
// Every micro-frontend (and the shell) externalises "vue" and resolves to this
// single module instance, guaranteeing one shared reactive runtime.
const { copyFileSync, mkdirSync } = require('node:fs')
const { dirname, resolve } = require('node:path')

const root = resolve(__dirname, '..')
const src = resolve(root, 'node_modules/vue/dist/vue.esm-browser.prod.js')
const dest = resolve(root, 'dist/mf/vue.js')

mkdirSync(dirname(dest), { recursive: true })
copyFileSync(src, dest)
console.log('shared vue ->', dest)
