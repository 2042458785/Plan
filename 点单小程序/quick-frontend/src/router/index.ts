import {createRouter,createWebHistory} from 'vue-router'
import QuickStorefrontView from '@/views/QuickStorefrontView.vue'
import NotFoundView from '@/views/NotFoundView.vue'
export default createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {path: '/', component: QuickStorefrontView},
    {path: '/:pathMatch(.*)*', component: NotFoundView},
  ],
})
