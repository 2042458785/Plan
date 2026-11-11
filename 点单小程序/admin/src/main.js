import Vue from 'vue'
import ElementUI from 'element-ui'
import 'element-ui/lib/theme-chalk/index.css'
import App from './App.vue'
import router from './router'
import store from './store'
import {logger} from './utils/logger'
import './styles/atelier.css'
Vue.config.productionTip=false
Vue.use(ElementUI)
Vue.config.errorHandler=()=>logger.error('main','errorHandler','商家页面运行异常')
new Vue({router,store,render:h=>h(App)}).$mount('#app')
