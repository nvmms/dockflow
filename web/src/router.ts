import { createRouter, createWebHistory } from 'vue-router'
import AppView from './views/App.vue'
import DatabaseView from './views/Database.vue'
import DeploymentView from './views/Deployment.vue'
import RedisView from './views/Redis.vue'
import RepositoryView from './views/Repository.vue'

export const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', redirect: '/apps' },
    { path: '/apps', name: 'apps', component: AppView, meta: { menu: 'apps' } },
    { path: '/apps/:app/deployments', name: 'deployments', component: DeploymentView, meta: { menu: 'apps' }, props: route => ({ app: route.params.app }) },
    { path: '/databases', name: 'databases', component: DatabaseView, meta: { menu: 'databases' } },
    { path: '/redis', name: 'redis', component: RedisView, meta: { menu: 'redis' } },
    { path: '/repositories', name: 'repositories', component: RepositoryView, meta: { menu: 'repositories' } },
    { path: '/:pathMatch(.*)*', redirect: '/apps' },
  ],
})
