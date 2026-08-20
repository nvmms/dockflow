<template>
  <section class="resource-page">
    <header class="page-header">
      <div><p class="eyebrow">DELIVERY</p><h1>{{ app }} 部署</h1><p class="page-description">查看该应用的部署任务、打包日志和运行日志。</p></div>
      <div class="header-actions"><el-button @click="$emit('back')">返回应用</el-button><el-button type="primary" @click="openDeploy">部署</el-button><el-button :loading="loading" @click="load">刷新</el-button></div>
    </header>
    <el-card shadow="never" class="resource-card">
      <div class="table-toolbar deploy-toolbar">
        <el-input v-model="search" placeholder="搜索任务 ID" clearable class="search-input" />
        <el-select v-model="status" class="status-filter">
          <el-option label="全部状态" value="all" />
          <el-option label="部署中" value="running" />
          <el-option label="成功" value="succeeded" />
          <el-option label="失败" value="failed" />
        </el-select>
        <span class="record-count">{{ filtered.length }} 个部署任务</span>
      </div>
      <el-table :data="filtered" v-loading="loading" empty-text="当前命名空间还没有部署任务">
        <el-table-column label="部署任务" min-width="190"><template #default="{ row }"><div class="primary-cell"><span class="resource-avatar deploy-avatar">D</span><div><strong>{{ row.id.slice(0, 12) }}</strong><small>{{ row.id }}</small></div></div></template></el-table-column>
        <el-table-column label="状态" width="110"><template #default="{ row }"><el-tag :type="tagType(row.status)" effect="plain">{{ statusText(row.status) }}</el-tag></template></el-table-column>
        <el-table-column label="开始时间" width="185"><template #default="{ row }">{{ formatTime(row.startedAt) }}</template></el-table-column>
        <el-table-column label="耗时" width="110"><template #default="{ row }">{{ duration(row) }}</template></el-table-column>
        <el-table-column label="结果" min-width="240"><template #default="{ row }"><span v-if="row.error" class="deployment-error" :title="row.error">{{ row.error }}</span><span v-else class="muted">{{ row.status === 'running' ? '正在执行' : '—' }}</span></template></el-table-column>
        <el-table-column align="right" width="250"><template #default="{ row }"><el-button link @click="openBuildLogs(row)">打包日志</el-button><el-button link :disabled="!latestContainer(row.app)" @click="openRuntimeLogs(row)">运行日志</el-button><el-button link type="danger" :disabled="row.status==='running'" @click="removeDeployment(row)">删除</el-button></template></el-table-column>
      </el-table>
    </el-card>
  </section>

  <el-dialog v-model="deployDialog" :title="`部署 ${app}`" width="480px">
    <el-form label-position="top">
      <el-form-item label="部署来源"><el-radio-group v-model="deployType"><el-radio-button value="branch">分支</el-radio-button><el-radio-button value="tag">标签</el-radio-button><el-radio-button value="commit">Commit</el-radio-button></el-radio-group></el-form-item>
      <el-form-item :label="deployType"><el-input v-model="deployValue" :placeholder="deployType === 'branch' ? 'main' : ''" /></el-form-item>
    </el-form>
    <template #footer><el-button @click="deployDialog = false">取消</el-button><el-button type="primary" :loading="deploying" @click="deploy">开始部署</el-button></template>
  </el-dialog>

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
import { ElMessage, ElMessageBox } from 'element-plus'
import { api, type AppRecord, type DeploymentJob } from '../api'

const props = defineProps<{ namespace: string; app: string }>()
defineEmits<{ back: [] }>()
const records = ref<DeploymentJob[]>([])
const apps = ref<AppRecord[]>([])
const loading = ref(false)
const search = ref('')
const status = ref('all')
const deployDialog = ref(false)
const deploying = ref(false)
const deployType = ref('branch')
const deployValue = ref('main')
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
  return matchesStatus && (!query || job.id.toLowerCase().includes(query))
}))

async function load() {
  loading.value = true
  try {
    const [jobs, appRecords] = await Promise.all([
      api.get<DeploymentJob[]>(`/namespaces/${props.namespace}/apps/${props.app}/deploy/jobs`),
      api.get<AppRecord[]>(`/namespaces/${props.namespace}/apps`),
    ])
    records.value = (jobs || []).sort((a, b) => b.startedAt.localeCompare(a.startedAt))
    apps.value = appRecords || []
    if (selected.value) selected.value = records.value.find(job => job.id === selected.value?.id)
    updatePolling()
  } catch (error) { ElMessage.error((error as Error).message) }
  finally { loading.value = false }
}
const selectedApp = computed(() => apps.value.find(app => app.name === selected.value?.app))
function openDeploy() {
  const current = apps.value.find(item => item.name === props.app)
  deployType.value = current?.trigger?.type || 'branch'
  deployValue.value = current?.trigger?.rule || 'main'
  deployDialog.value = true
}
async function deploy() {
  if (!deployValue.value.trim()) return ElMessage.warning('请输入部署来源')
  deploying.value = true
  try {
    const job = await api.post<DeploymentJob>(`/namespaces/${props.namespace}/apps/${props.app}/deploy`, { [deployType.value]: deployValue.value.trim() })
    deployDialog.value = false
    selected.value = job
    buildLogsDialog.value = true
    await load()
    ElMessage.success('部署任务已进入后台')
  } catch (error) { ElMessage.error((error as Error).message) }
  finally { deploying.value = false }
}
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
async function removeDeployment(job: DeploymentJob) {
  try {
    await ElMessageBox.confirm(`确定删除部署任务 “${job.id.slice(0, 12)}” 及其日志记录吗？`, '删除部署记录', { type: 'warning' })
    await api.delete(`/namespaces/${props.namespace}/apps/${props.app}/deploy/jobs/${job.id}`)
    if (selected.value?.id === job.id) { selected.value = undefined; buildLogsDialog.value = false; runtimeLogsDialog.value = false }
    ElMessage.success('部署记录已删除')
    await load()
  } catch (error) { if (error !== 'cancel' && error !== 'close') ElMessage.error((error as Error).message) }
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

watch(() => [props.namespace, props.app], load, { immediate: true })
onBeforeUnmount(() => { if (pollTimer !== undefined) window.clearInterval(pollTimer) })
</script>

<style scoped>
.deploy-toolbar { justify-content: flex-start; gap: 12px; }
.deploy-toolbar .record-count { margin-left: auto; }
.header-actions { display: flex; gap: 10px; }
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
