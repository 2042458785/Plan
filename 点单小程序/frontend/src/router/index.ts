import {createRouter,createWebHistory} from 'vue-router'
import StorefrontView from '@/views/StorefrontView.vue'
import NotFoundView from '@/views/NotFoundView.vue'
export default createRouter({
  history: createWebHistory(),
  routes: [
    {path: '/', component: StorefrontView},
    {path: '/:pathMatch(.*)*', component: NotFoundView},
  ],
})
