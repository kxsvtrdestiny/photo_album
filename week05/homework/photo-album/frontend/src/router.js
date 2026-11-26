import { createRouter, createWebHistory } from 'vue-router'
import { session } from './session'
import AccountView from './views/AccountView.vue'
import AlbumsView from './views/AlbumsView.vue'
import PhotosView from './views/PhotosView.vue'
import NotFoundView from './views/NotFoundView.vue'

export const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', name: 'public', component: AlbumsView, props: { mine: false } },
    { path: '/mine', name: 'mine', component: AlbumsView, props: { mine: true }, meta: { protected: true } },
    { path: '/albums/:id', name: 'photos', component: PhotosView, props: true },
    { path: '/login', name: 'login', component: AccountView, props: { register: false } },
    { path: '/register', name: 'register', component: AccountView, props: { register: true } },
    { path: '/:pathMatch(.*)*', component: NotFoundView },
  ],
})

router.beforeEach((to) => {
  if (to.meta.protected && !session.value) {
    return { name: 'login', query: { redirect: to.fullPath } }
  }
  if ((to.name === 'login' || to.name === 'register') && session.value) return { name: 'mine' }
})
