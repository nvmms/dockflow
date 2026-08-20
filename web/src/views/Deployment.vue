<template>
  <section class="resource-page">
    <header class="page-header">
      <div><p class="eyebrow">DELIVERY</p><h1>部署</h1><p class="page-description">查看当前命名空间中所有应用的部署任务与执行结果。</p></div>
      <el-button :loading="loading" @click="load">刷新</el-button>
    </header>
    <el-card shadow="never" class="resource-card">
      <div class="table-toolbar deploy-toolbar">
        <el-input v-model="search" placeholder="搜索应用或任务 ID" clearable class="search-input" />
        <el-select v-model="status" class="status-filter">
          <el-option label="全部状态" value="all" />
          <el-option label="部署中" value="running" />
          <el-option label="成功" value="succeeded" />
          <el-option label="失败" value="failed" />
        </el-select>
        <span class="record-count">{{ filtered.length }} 个部署任务</span>
      </div>
      <el-table :data="filtered" v-loading="loading" empty-text="当前命名空间还没有部署任务">
        <el-table-column label="应用" min-width="190"><template #default="{ row }"><div class="primary-cell"><span class="resource-avatar deploy-avatar">D</span><div><strong>{{ row.app }}</strong><small>{{ row.id }}</small></div></div></template></el-table-column>
        <el-table-column label="状态" width="110"><template #default="{ row }"><el-tag :type="tagType(row.status)" effect="plain">{{ statusText(row.status) }}</el-tag></template></el-table-column>
        <el-table-column label="开始时间" width="185"><template #default="{ row }">{{ formatTime(row.startedAt) }}</template></el-table-column>
        <el-table-column label="耗时" width="110"><template #default="{ row }">{{ duration(row) }}</template></el-table-column>
        <el-table-column label="结果" min-width="240"><template #default="{ row }"><span v-if="row.error" class="deployment-error" :title="row.error">{{ row.error }}</span><span v-else class="muted">{{ row.status === 'running' ? '正在执行' : '—' }}</span></template></el-table-column>
        <el-table-column align="right" width="190"><template #default="{ row }"><el-button link @click="openBuildLogs(row)">打包日志</el-button><el-button link :disabled="!latestContainer(row.app)" @click="openRuntimeLogs(row)">运行日志</el-button></template></el-table-column>
      </el-table>
    </el-card>
  </section>

  <el-dialog v-model="buildLogsDialog" :title="`${selected?.app || ''} 打包日志`" width="900px">
    <div class="log-toolbar"><el-tag :type="tagType(selected?.status)" effect="dark">{{ statusText(selected?.status) }}</el-tag><span class="job-id">任务 {{ selected?.id }}</span><el-button v-if="selected?.status === 'running'" :loading="loading" @click="load">刷新</el-button></div>
    <pre class="log-output">{{ selected?.logs || '暂无部署日志' }}</pre>
    <el-alert v-if="selected?.error" :title="selected.error" type="error" :closable="false" show-icon class="deploy-error-alert" />
  </el-dialog>

  <el-dialog v-model="runtimeLogsDialog" :title="`${selected?.app || ''} 运行日志`" width="900px">
    <div class="log-toolbar">
      <el-select v-model="selectedContainer" @change="loadRuntimeLogs">
        <el-option v-for="item in selectedApp?.deploy || []" :key="item.containerId" :label="`${item.version} · ${item.containerId.slice(0, 12)}`" :value="item.containerId" />
      </el-select>
      <el-select v-model="logTail" class="tail-select" @change="loadRuntimeLogs"><el-option label="最近 100 行" value="100"/><el-option label="最近 200 行" value="200"/><el-option label="最近 500 行" value="500"/><el-option label="最近 1000 行" value="1000"/></el-select>
      <el-button :loading="runtimeLogsLoading" @click="loadRuntimeLogs">刷新</el-button>
    </div>
    <pre v-loading="runtimeLogsLoading" class="log-output">{{ runtimeLogs || '暂无运行日志' }}</pre>
  </el-dialog>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { api, type AppRecord, type DeploymentJob } from '../api'

const props = defineProps<{ namespace: string }>()
const records = ref<DeploymentJob[]>([])
const apps = ref<AppRecord[]>([])
const loading = ref(false)
const search = ref('')
const status = ref('all')
const buildLogsDialog = ref(false)
const runtimeLogsDialog = ref(false)
const runtimeLogsLoading = ref(false)
const selectedContainer = ref('')
const logTail = ref('200')
const runtimeLogs = ref('')
const selected = ref<DeploymentJob>()
let pollTimer: number | undefined

const filtered = computed(() => records.value.filter(job => {
  const matchesStatus = status.value === 'all' || job.status === status.value
  const query = search.value.trim().toLowerCase()
  return matchesStatus && (!query || `${job.app} ${job.id}`.toLowerCase().includes(query))
}))

async function load() {
  loading.value = true
  try {
    const [jobs, appRecords] = await Promise.all([
      api.get<DeploymentJob[]>(`/namespaces/${props.namespace}/deployments`),
      api.get<AppRecord[]>(`/namespaces/${props.namespace}/apps`),
    ])
    records.value = jobs || []
    apps.value = appRecords || []
    if (selected.value) selected.value = records.value.find(job => job.id === selected.value?.id)
    updatePolling()
  } catch (error) { ElMessage.error((error as Error).message) }
  finally { loading.value = false }
}
const selectedApp = computed(() => apps.value.find(app => app.name === selected.value?.app))
function latestContainer(appName: string) { return apps.value.find(app => app.name === appName)?.deploy?.at(-1)?.containerId || '' }
function openBuildLogs(job: DeploymentJob) { selected.value = job; buildLogsDialog.value = true }
function openRuntimeLogs(job: DeploymentJob) {
  selected.value = job
  selectedContainer.value = latestContainer(job.app)
  runtimeLogs.value = ''
  runtimeLogsDialog.value = true
  loadRuntimeLogs()
}
async function loadRuntimeLogs() {
  if (!selected.value || !selectedContainer.value) return
  runtimeLogsLoading.value = true
  try {
    const result = await api.get<{ logs: string }>(`/namespaces/${props.namespace}/apps/${selected.value.app}/deploy/${selectedContainer.value}/logs?tail=${logTail.value}`)
    runtimeLogs.value = result.logs
  } catch (error) { runtimeLogs.value = ''; ElMessage.error((error as Error).message) }
  finally { runtimeLogsLoading.value = false }
}
function updatePolling() {
  const hasRunning = records.value.some(job => job.status === 'running')
  if (hasRunning && pollTimer === undefined) pollTimer = window.setInterval(load, 2000)
  if (!hasRunning && pollTimer !== undefined) { window.clearInterval(pollTimer); pollTimer = undefined }
}
function tagType(value?: string) { return value === 'succeeded' ? 'success' : value === 'failed' ? 'danger' : 'warning' }
function statusText(value?: string) { return value === 'succeeded' ? '成功' : value === 'failed' ? '失败' : value === 'running' ? '部署中' : '未知' }
function formatTime(value: string) { return new Date(value).toLocaleString() }
function duration(job: DeploymentJob) {
  const end = job.finishedAt ? new Date(job.finishedAt).getTime() : Date.now()
  const seconds = Math.max(0, Math.round((end - new Date(job.startedAt).getTime()) / 1000))
  return seconds < 60 ? `${seconds} 秒` : `${Math.floor(seconds / 60)}分 ${seconds % 60}秒`
}

watch(() => props.namespace, load, { immediate: true })
onBeforeUnmount(() => { if (pollTimer !== undefined) window.clearInterval(pollTimer) })
</script>

<style scoped>
.deploy-toolbar { justify-content: flex-start; gap: 12px; }
.deploy-toolbar .record-count { margin-left: auto; }
.status-filter { width: 140px; }
.deploy-avatar { color: #25785f; background: #e5f7f0; }
.deployment-error { display: block; overflow: hidden; color: #c04444; text-overflow: ellipsis; white-space: nowrap; }
.log-toolbar { display: flex; align-items: center; gap: 10px; margin-bottom: 12px; }
.log-toolbar .el-button { margin-left: auto; }
.job-id { color: #8b94a5; font-size: 12px; }
.tail-select { width: 150px; }
.log-output { min-height: 440px; max-height: 60vh; margin: 0; padding: 16px; overflow: auto; border-radius: 8px; background: #101521; color: #d8e0ee; font: 12px/1.6 ui-monospace, SFMono-Regular, Consolas, monospace; white-space: pre-wrap; word-break: break-all; }
.deploy-error-alert { margin-top: 12px; }
</style>
