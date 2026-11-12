import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
const project=path.resolve(import.meta.dirname,'..')
const read=file=>fs.readFileSync(path.join(project,file),'utf8')
const customer=JSON.parse(read('frontend/package.json')),admin=JSON.parse(read('admin/package.json'))
const quick=JSON.parse(read('quick-frontend/package.json'))
assert.equal(customer.dependencies.vue,'3.5.39')
assert.equal(customer.scripts.serve,'vite')
assert.equal(quick.scripts.serve,'vite')
assert.equal(quick.scripts.build,'vite build')
assert.equal(admin.dependencies.vue,'^2.6.14')
assert.equal(admin.scripts.serve,'vue-cli-service serve')
assert.ok(customer.dependencies.axios&&customer.dependencies.vuex&&customer.dependencies['vue-router'])
assert.ok(admin.dependencies.axios&&admin.dependencies.vuex&&admin.dependencies['vue-router']&&admin.dependencies['element-ui'])
assert.ok(!fs.existsSync(path.join(project,'frontend/src/Admin.vue')))
for(const file of ['frontend/src/config.ts','frontend/src/router/index.ts','frontend/src/store/index.ts','frontend/src/utils/request.ts','admin/src/config.js','admin/src/router/index.js','admin/src/store/index.js','admin/src/utils/request.js','backend/src/config/config.example.yaml','backend/src/app/app.go','backend/src/dao/order_dao.go','backend/src/api/http_controller/order_controller.go'])assert.ok(fs.existsSync(path.join(project,file)),`缺少架构文件 ${file}`)
const marker=['商家入口','merchant-token','/admin/login','/api/v1/admin','MERCHANT ATELIER','suxu:unauthorized']
function walk(directory){return fs.readdirSync(directory,{withFileTypes:true}).flatMap(entry=>entry.isDirectory()?walk(path.join(directory,entry.name)):[path.join(directory,entry.name)])}
const bundleFiles=['frontend/dist','quick-frontend/quick'].flatMap(directory=>{
  assert.ok(fs.existsSync(path.join(project,directory,'index.html')),`缺少独立产物 ${directory}/index.html`)
  return walk(path.join(project,directory)).filter(file=>/\.(js|html|css)$/.test(file))
})
for(const file of bundleFiles){const text=fs.readFileSync(file,'utf8');for(const value of marker){assert.ok(!text.includes(value),`顾客产物泄漏管理端内容：${path.basename(file)} / ${value}`)}}
assert.ok(fs.existsSync(path.join(project,'admin/dist_end/index.html')),'管理端需独立构建到 dist_end')
assert.ok(read('backend/src/config/mysql.go').includes('AutoMigrate'))
assert.ok(!read('backend/src/config/mysql.go').includes('Snapshot'))
assert.ok(read('backend/src/dao/order_dao.go').includes('WithTransaction'))
assert.ok(read('backend/src/middleware/auth.go').includes('auth.Validate'))
process.stdout.write(`PASS：独立框架、入口与 ${bundleFiles.length} 个顾客产物的后台隔离检查。\n`)
