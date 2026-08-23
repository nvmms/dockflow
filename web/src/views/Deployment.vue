<template>
  <section class="resource-page">
    <header class="page-header">
      <div><p class="eyebrow">DELIVERY</p><h1>{{ app }} 部署</h1><p class="page-description">查看该应用的部署任务、打包日志和运行日志。</p></div>
      <div class="header-actions"><el-button @click="$emit('back')">返回应用</el-button><el-button type="primary" @click="openDeploy">部署</el-button><el-button :loading="loading" @click="load">刷新</el-button></div>
    </header>
    <el-card shadow="never" class="resource-card">
      <div class="table-toolbar deploy-toolbar">
        <el-input v-model="search" placeholder="搜索分支、版本、Commit 或任务 ID" clearable class="search-input" />
        <el-select v-model="status" class="status-filter">
          <el-option label="全部状态" value="all" />
          <el-option label="运行中" value="running" />
          <el-option label="成功" value="success" />
          <el-option label="失败" value="failed" />
          <el-option label="停止" value="stopped" />
        </el-select>
        <span class="record-count">{{ filtered.length }} 个部署任务</span>
      </div>
      <el-table :data="filtered" v-loading="loading" empty-text="当前命名空间还没有部署任务">
        <el-table-column label="部署版本" min-width="255"><template #default="{ row }"><div class="primary-cell"><span class="resource-avatar deploy-avatar">D</span><div class="deployment-identity"><div class="deployment-title"><strong>{{ deploymentTitle(row) }}</strong><el-tag v-if="row.sourceType" size="small" effect="plain" round>{{ sourceTypeText(row.sourceType) }}</el-tag><el-tag v-if="row.needs_recreate" type="warning" size="small" effect="plain">需重建</el-tag></div><small>{{ deploymentDetail(row) }}</small></div></div></template></el-table-column>
        <el-table-column label="状态" width="110"><template #default="{ row }"><el-tag :type="tagType(row.status)" effect="plain">{{ statusText(row.status) }}</el-tag></template></el-table-column>
        <el-table-column label="访问域名" min-width="210"><template #default="{ row }"><a v-if="row.domain" class="domain-link" :href="domainURL(row.domain)" target="_blank" rel="noopener noreferrer">{{ row.domain }}</a><span v-else class="muted">—</span></template></el-table-column>
        <el-table-column label="IP" min-width="150"><template #default="{ row }"><span v-if="row.ip?.length">{{ row.ip.join(', ') }}</span><span v-else class="muted">—</span></template></el-table-column>
        <el-table-column label="创建时间" width="185"><template #default="{ row }">{{ formatTime(row.startedAt) }}</template></el-table-column>
        <el-table-column label="结果" min-width="240"><template #default="{ row }"><span v-if="row.error" class="deployment-error" :title="row.error">{{ row.error }}</span><span v-else class="muted">{{ row.status === 'running' ? '正在执行' : '—' }}</span></template></el-table-column>
        <el-table-column align="right" width="350"><template #default="{ row }"><el-button link :disabled="!row.containerId || row.status === 'running'" @click="openEdit(row)">编辑</el-button><el-button link @click="openBuildLogs(row)">打包日志</el-button><el-button link :disabled="!row.containerId" @click="openRuntimeLogs(row)">运行日志</el-button><el-button link type="primary" :disabled="!row.containerId || row.status === 'running'" @click="restartDeployment(row)">重启</el-button><el-button link type="danger" :disabled="row.status==='running'" @click="removeDeployment(row)">删除</el-button></template></el-table-column>
      </el-table>
    </el-card>
  </section>

  <el-dialog v-model="deployDialog" :title="`部署 ${app}`" width="480px">
    <el-form label-position="top">
      <el-form-item label="部署来源"><el-radio-group v-model="deployType"><el-radio-button value="branch">分支</el-radio-button><el-radio-button value="tag">标签</el-radio-button><el-radio-button value="commit">Commit</el-radio-button></el-radio-group></el-form-item>
      <el-form-item :label="deployType"><el-input v-model="deployValue" :placeholder="deployType === 'branch' ? 'main' : ''" /></el-form-item>
      <el-form-item label="访问域名" required><el-input v-model="deployDomain" placeholder="app.example.com" /><div class="form-hint">每次部署可填写不同域名，无需包含 http:// 或 https://。</div></el-form-item>
      <el-form-item label="自动重启策略"><el-select v-model="deployRestartPolicy"><el-option label="除非手动停止（推荐）" value="unless-stopped"/><el-option label="始终自动重启" value="always"/><el-option label="仅失败时重启" value="on-failure"/><el-option label="不自动重启" value="no"/></el-select></el-form-item>
      <el-divider>日志策略</el-divider>
      <el-form-item label="日志驱动"><el-select v-model="deployLogDriver"><el-option label="local（推荐）" value="local"/><el-option label="json-file" value="json-file"/><el-option label="阿里云日志服务" value="aliyun-sls"/></el-select></el-form-item>
      <div v-if="deployLogDriver !== 'aliyun-sls'" class="form-grid"><el-form-item label="单文件上限"><el-input v-model="deployLogMaxSize" placeholder="10m"/></el-form-item><el-form-item label="保留文件数"><el-input-number v-model="deployLogMaxFile" :min="1" :max="100"/></el-form-item></div>
      <div v-else class="form-grid"><el-form-item label="Endpoint"><el-input v-model="deploySLSEndpoint"/></el-form-item><el-form-item label="Project"><el-input v-model="deploySLSProject"/></el-form-item><el-form-item label="Logstore"><el-input v-model="deploySLSLogstore"/></el-form-item><el-form-item label="采集配置名称"><el-input v-model="deploySLSConfigName"/></el-form-item></div>
    </el-form>
    <template #footer><el-button @click="deployDialog = false">取消</el-button><el-button type="primary" :loading="deploying" @click="deploy">开始部署</el-button></template>
  </el-dialog>

  <el-dialog v-model="editDialog" title="编辑部署配置" width="520px">
    <el-form label-position="top">
      <el-form-item label="自动重启策略"><el-select v-model="editRestartPolicy"><el-option label="除非手动停止（推荐）" value="unless-stopped"/><el-option label="始终自动重启" value="always"/><el-option label="仅失败时重启" value="on-failure"/><el-option label="不自动重启" value="no"/></el-select></el-form-item>
      <el-divider>日志策略</el-divider>
      <el-form-item label="日志驱动"><el-select v-model="editLogDriver"><el-option label="local（推荐）" value="local"/><el-option label="json-file" value="json-file"/><el-option label="阿里云日志服务" value="aliyun-sls"/></el-select></el-form-item>
      <div v-if="editLogDriver !== 'aliyun-sls'" class="form-grid"><el-form-item label="单文件上限"><el-input v-model="editLogMaxSize"/></el-form-item><el-form-item label="保留文件数"><el-input-number v-model="editLogMaxFile" :min="1" :max="100"/></el-form-item></div>
      <div v-else class="form-grid"><el-form-item label="Endpoint"><el-input v-model="editSLSEndpoint"/></el-form-item><el-form-item label="Project"><el-input v-model="editSLSProject"/></el-form-item><el-form-item label="Logstore"><el-input v-model="editSLSLogstore"/></el-form-item><el-form-item label="采集配置名称"><el-input v-model="editSLSConfigName"/></el-form-item></div>
      <el-form-item label="立即应用"><el-switch v-model="editApplyNow"/><div class="form-hint">开启后停止并按新配置重建该部署容器；关闭时保存配置并标记“需重建”。</div></el-form-item>
    </el-form>
    <template #footer><el-button @click="editDialog=false">取消</el-button><el-button type="primary" :loading="editSaving" @click="saveEdit">保存</el-button></template>
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
import { openLogStream, type LogStream } from '../logStream'

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
const deployDomain = ref('')
const deployRestartPolicy = ref('unless-stopped')
const deployLogDriver = ref('local')
const deployLogMaxSize = ref('10m')
const deployLogMaxFile = ref(3)
const deploySLSProject = ref('')
const deploySLSLogstore = ref('')
const deploySLSEndpoint = ref('')
const deploySLSConfigName = ref('')
const editDialog = ref(false)
const editSaving = ref(false)
const editTarget = ref<DeploymentJob>()
const editRestartPolicy = ref('unless-stopped')
const editLogDriver = ref('local')
const editLogMaxSize = ref('10m')
const editLogMaxFile = ref(3)
const editApplyNow = ref(false)
const editSLSProject = ref('')
const editSLSLogstore = ref('')
const editSLSEndpoint = ref('')
const editSLSConfigName = ref('')
const buildLogsDialog = ref(false)
const runtimeLogsDialog = ref(false)
const runtimeLogsLoading = ref(false)
const selectedContainer = ref('')
const logTail = ref('200')
const runtimeLogs = ref('')
const selected = ref<DeploymentJob>()
let pollTimer: number | undefined
let buildLogStream: LogStream | undefined
let runtimeLogStream: LogStream | undefined

const filtered = computed(() => records.value.filter(job => {
  const matchesStatus = status.value === 'all' || job.status === status.value
  const query = search.value.trim().toLowerCase()
  const searchable = [job.id, job.sourceType, job.sourceRef, job.commit, job.version, job.containerId, job.domain].filter(Boolean).join(' ').toLowerCase()
  return matchesStatus && (!query || searchable.includes(query))
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
  deployDomain.value = current?.url?.[0]?.host || ''
  deployRestartPolicy.value = 'unless-stopped'
  deployLogDriver.value = 'local'
  deployLogMaxSize.value = '10m'
  deployLogMaxFile.value = 3
  deployDialog.value = true
}
async function deploy() {
  if (!deployValue.value.trim()) return ElMessage.warning('请输入部署来源')
  if (!deployDomain.value.trim()) return ElMessage.warning('请输入访问域名')
  deploying.value = true
  try {
    const job = await api.post<DeploymentJob>(`/namespaces/${props.namespace}/apps/${props.app}/deploy`, { [deployType.value]: deployValue.value.trim(), domain: deployDomain.value.trim(), restart_policy: deployRestartPolicy.value, log_driver: deployLogDriver.value, log_max_size: deployLogMaxSize.value, log_max_file: deployLogMaxFile.value, sls_project: deploySLSProject.value, sls_logstore: deploySLSLogstore.value, sls_endpoint: deploySLSEndpoint.value, sls_config_name: deploySLSConfigName.value })
    deployDialog.value = false
    openBuildLogs(job)
    await load()
    ElMessage.success('部署任务已进入后台')
  } catch (error) { ElMessage.error((error as Error).message) }
  finally { deploying.value = false }
}
function latestContainer(appName: string) { return apps.value.find(app => app.name === appName)?.deploy?.at(-1)?.containerId || '' }
function openBuildLogs(job: DeploymentJob) {
  selected.value = job
  buildLogsDialog.value = true
  buildLogStream?.close()
  selected.value.logs = ''
  let streamedLogs = ''
  buildLogStream = openLogStream(`/namespaces/${props.namespace}/apps/${props.app}/deploy/jobs/${job.id}/logs`, {
    onLog: chunk => { streamedLogs += chunk; if (selected.value?.id === job.id) selected.value.logs = streamedLogs },
    onDone: () => load(),
    onError: () => ElMessage.error('部署日志连接已断开'),
  })
}
function openEdit(job: DeploymentJob) {
  editTarget.value = job
  editRestartPolicy.value = job.restart_policy || 'unless-stopped'
  editLogDriver.value = job.log_driver || 'local'
  editLogMaxSize.value = job.log_max_size || '10m'
  editLogMaxFile.value = job.log_max_file || 3
  editSLSProject.value = job.sls_project || ''
  editSLSLogstore.value = job.sls_logstore || ''
  editSLSEndpoint.value = job.sls_endpoint || ''
  editSLSConfigName.value = job.sls_config_name || ''
  editApplyNow.value = false
  editDialog.value = true
}
async function saveEdit() {
  if (!editTarget.value) return
  editSaving.value = true
  try {
    await api.put(`/namespaces/${props.namespace}/apps/${props.app}/deploy/jobs/${editTarget.value.id}`, {restart_policy: editRestartPolicy.value, log_driver: editLogDriver.value, log_max_size: editLogMaxSize.value, log_max_file: editLogMaxFile.value, apply_now: editApplyNow.value, sls_project: editSLSProject.value, sls_logstore: editSLSLogstore.value, sls_endpoint: editSLSEndpoint.value, sls_config_name: editSLSConfigName.value})
    editDialog.value = false
    ElMessage.success(editApplyNow.value ? '部署已按新配置重建' : '配置已保存，等待重建')
    await load()
  } catch (error) { ElMessage.error((error as Error).message) }
  finally { editSaving.value = false }
}
function openRuntimeLogs(job: DeploymentJob) {
  selected.value = job
  selectedContainer.value = job.containerId || latestContainer(job.app)
  runtimeLogs.value = ''
  runtimeLogsDialog.value = true
  loadRuntimeLogs()
}
async function restartDeployment(job: DeploymentJob) {
  try {
    await ElMessageBox.confirm(`确定重启部署“${deploymentTitle(job)}”的容器吗？`, '重启部署', { type: 'warning' })
    await api.post<DeploymentJob>(`/namespaces/${props.namespace}/apps/${props.app}/deploy/jobs/${job.id}/restart`)
    ElMessage.success('部署已重启')
    await load()
  } catch (error) { if (error !== 'cancel' && error !== 'close') ElMessage.error((error as Error).message) }
}
async function loadRuntimeLogs() {
  if (!selected.value || !selectedContainer.value) return
  runtimeLogStream?.close()
  runtimeLogs.value = ''
  runtimeLogsLoading.value = true
  runtimeLogStream = openLogStream(`/namespaces/${props.namespace}/apps/${selected.value.app}/deploy/${selectedContainer.value}/logs/stream?tail=${logTail.value}`, {
    onLog: chunk => { runtimeLogsLoading.value = false; runtimeLogs.value += chunk },
    onDone: () => { runtimeLogsLoading.value = false },
    onError: () => { runtimeLogsLoading.value = false; ElMessage.error('运行日志连接已断开') },
  })
}
async function removeDeployment(job: DeploymentJob) {
  try {
    await ElMessageBox.confirm(`确定删除部署“${deploymentTitle(job)}”吗？对应容器、运行日志和 Traefik 路由也会被删除。`, '删除部署', { type: 'warning' })
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
function tagType(value?: string) { return value === 'success' ? 'success' : value === 'failed' ? 'danger' : value === 'stopped' ? 'info' : 'warning' }
function statusText(value?: string) { return value === 'success' ? '成功' : value === 'failed' ? '失败' : value === 'running' ? '运行中' : value === 'stopped' ? '停止' : '未知' }
function sourceTypeText(value?: string) { return value === 'branch' ? '分支' : value === 'tag' ? '标签' : value === 'commit' ? 'Commit' : '历史' }
function shortRevision(value?: string) { return value ? value.slice(0, 12) : '' }
function deploymentTitle(job: DeploymentJob) {
  if (job.sourceType === 'branch' || job.sourceType === 'tag') return job.sourceRef || job.version || '未知来源'
  if (job.sourceType === 'commit') return shortRevision(job.sourceRef || job.commit)
  return job.version ? shortRevision(job.version) : job.sourceRef || `任务 ${job.id.slice(0, 12)}`
}
function deploymentDetail(job: DeploymentJob) {
  const revision = shortRevision(job.version || job.commit)
  const parts = []
  if (revision && revision !== deploymentTitle(job)) parts.push(`版本 ${revision}`)
  parts.push(`任务 ${job.id.slice(0, 12)}`)
  return parts.join(' · ')
}
function formatTime(value: string) {
  const time = new Date(value)
  return Number.isNaN(time.getTime()) || time.getFullYear() <= 1 ? '—' : time.toLocaleString()
}
function domainURL(domain: string) { return `https://${domain}` }
watch(() => [props.namespace, props.app], load, { immediate: true })
watch(buildLogsDialog, open => { if (!open) { buildLogStream?.close(); buildLogStream = undefined } })
watch(runtimeLogsDialog, open => { if (!open) { runtimeLogStream?.close(); runtimeLogStream = undefined } })
onBeforeUnmount(() => { if (pollTimer !== undefined) window.clearInterval(pollTimer); buildLogStream?.close(); runtimeLogStream?.close() })
</script>

<style scoped>
.deploy-toolbar { justify-content: flex-start; gap: 12px; }
.deploy-toolbar .record-count { margin-left: auto; }
.header-actions { display: flex; gap: 10px; }
.status-filter { width: 140px; }
.deploy-avatar { color: #25785f; background: #e5f7f0; }
.deployment-identity { min-width: 0; }
.deployment-title { display: flex; align-items: center; gap: 8px; min-width: 0; }
.deployment-title strong { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.deployment-error { display: block; overflow: hidden; color: #c04444; text-overflow: ellipsis; white-space: nowrap; }
.domain-link { color: var(--el-color-primary); text-decoration: none; }
.domain-link:hover { text-decoration: underline; }
.log-toolbar { display: flex; align-items: center; gap: 10px; margin-bottom: 12px; }
.log-toolbar .el-button { margin-left: auto; }
.job-id { color: #8b94a5; font-size: 12px; }
.tail-select { width: 150px; }
.log-output { min-height: 440px; max-height: 60vh; margin: 0; padding: 16px; overflow: auto; border-radius: 8px; background: #101521; color: #d8e0ee; font: 12px/1.6 ui-monospace, SFMono-Regular, Consolas, monospace; white-space: pre-wrap; word-break: break-all; }
.deploy-error-alert { margin-top: 12px; }
</style>
