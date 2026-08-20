<template>
  <el-container class="app-shell">
    <el-header class="topbar">
      <div class="brand"><span class="brand-mark">D</span><span>DockFlow</span></div>
      <div class="namespace-switcher">
        <span class="context-label">命名空间</span>
        <el-select v-model="currentNamespace" placeholder="选择命名空间" :loading="loadingNamespaces" @change="loadNamespaceDetail">
          <el-option v-for="item in namespaces" :key="item.name" :label="item.name" :value="item.name" />
        </el-select>
        <el-button text title="刷新" @click="loadNamespaces">↻</el-button>
        <el-button type="primary" plain @click="namespaceDialog = true">新建</el-button>
      </div>
      <div class="topbar-status"><span class="status-dot"></span> API 已连接</div>
    </el-header>
    <el-container class="body-shell">
      <el-aside width="216px" class="sidebar">
        <Menu v-model="activeView" />
        <div class="sidebar-footer">
          <div class="version">DockFlow Console</div>
          <a href="/api/v1/openapi.json" target="_blank">API 文档 ↗</a>
        </div>
      </el-aside>
      <el-main class="main-content">
        <div v-if="!currentNamespace" class="empty-state">
          <div class="empty-symbol">⌁</div>
          <h2>先创建一个命名空间</h2>
          <p>命名空间用于隔离应用、数据库和 Redis 资源。</p>
          <el-button type="primary" @click="namespaceDialog = true">创建命名空间</el-button>
        </div>
        <template v-else>
          <AppView v-if="activeView === 'apps'" :namespace="currentNamespace" />
          <DeploymentView v-else-if="activeView === 'deployments'" :namespace="currentNamespace" />
          <DatabaseView v-else-if="activeView === 'databases'" :namespace="currentNamespace" />
          <RedisView v-else-if="activeView === 'redis'" :namespace="currentNamespace" />
          <RepositoryView v-else />
        </template>
      </el-main>
    </el-container>
  </el-container>

  <el-dialog v-model="namespaceDialog" title="创建命名空间" width="440px" destroy-on-close>
    <el-form label-position="top" @submit.prevent="createNamespace">
      <el-form-item label="名称" required>
        <el-input v-model="newNamespace" placeholder="例如 production" autofocus @keyup.enter="createNamespace" />
      </el-form-item>
      <div class="form-hint">将自动创建独立的 Docker 网络和可用子网。</div>
    </el-form>
    <template #footer><el-button @click="namespaceDialog = false">取消</el-button><el-button type="primary" :loading="creating" @click="createNamespace">创建</el-button></template>
  </el-dialog>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { api, type Namespace } from './api'
import Menu from './components/Menu.vue'
import AppView from './views/App.vue'
import DeploymentView from './views/Deployment.vue'
import DatabaseView from './views/Database.vue'
import RedisView from './views/Redis.vue'
import RepositoryView from './views/Repository.vue'

const activeView = ref('apps')
const namespaces = ref<Namespace[]>([])
const currentNamespace = ref('')
const loadingNamespaces = ref(false)
const namespaceDialog = ref(false)
const newNamespace = ref('')
const creating = ref(false)

async function loadNamespaces() {
  loadingNamespaces.value = true
  try {
    namespaces.value = await api.get<Namespace[]>('/namespaces') || []
    if (!namespaces.value.some(n => n.name === currentNamespace.value)) currentNamespace.value = namespaces.value[0]?.name || ''
  } catch (error) { ElMessage.error((error as Error).message) }
  finally { loadingNamespaces.value = false }
}
function loadNamespaceDetail() { /* child views react to namespace changes */ }
async function createNamespace() {
  if (!newNamespace.value.trim()) return ElMessage.warning('请输入命名空间名称')
  creating.value = true
  try {
    const created = await api.post<Namespace>('/namespaces', { name: newNamespace.value.trim() })
    namespaceDialog.value = false; newNamespace.value = ''; await loadNamespaces(); currentNamespace.value = created.name
    ElMessage.success('命名空间已创建')
  } catch (error) { ElMessage.error((error as Error).message) }
  finally { creating.value = false }
}
onMounted(loadNamespaces)
</script>
