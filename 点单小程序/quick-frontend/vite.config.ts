import {defineConfig} from 'vite'
import vue from '@vitejs/plugin-vue'
import {fileURLToPath,URL} from 'node:url'
const backend={target:'http://127.0.0.1:8090',changeOrigin:false}
export default defineConfig({
 base:'/quick/',
 plugins:[vue(),{name:'storefront-route-boundary',configureServer(server){server.middlewares.use((req,res,next)=>{if(/^\/(?:admin(?:\/|$)|api\/v1\/admin(?:\/|$))/.test(req.url||'')){res.statusCode=404;res.end('Not found');return}next()})}}],
 resolve:{alias:{'@':fileURLToPath(new URL('./src',import.meta.url))}},
 server:{host:'0.0.0.0',port:8083,strictPort:true,proxy:{'/api/v1/catalog':backend,'/api/v1/orders':backend,'/api/v1/media':backend,'/api/v1/customers':backend}},
 build:{outDir:'quick',sourcemap:false},
})
