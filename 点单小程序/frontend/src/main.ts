import {createApp} from 'vue'
import {ElLoading} from 'element-plus'
import 'element-plus/es/components/loading/style/css'
import App from './App.vue'
import router from './router'
import store from './store'
import {logger} from './utils/logger'
import './styles/reset.css'
import './styles/storefront.css'
const app=createApp(App)
app.use(router).use(store).use(ElLoading)
app.config.errorHandler=()=>logger.error('main','errorHandler','页面运行异常')
app.mount('#app')
