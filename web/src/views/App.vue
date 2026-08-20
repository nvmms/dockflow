<template>
  <section class="resource-page">
    <header class="page-header"><div><p class="eyebrow">WORKLOADS</p><h1>应用</h1><p class="page-description">从 Git 仓库构建、部署和管理服务。</p></div><el-button type="primary" @click="openCreate">创建应用</el-button></header>
    <el-card shadow="never" class="resource-card">
      <div class="table-toolbar"><el-input v-model="search" placeholder="搜索应用或仓库" clearable class="search-input" /><el-button text :loading="loading" @click="load">刷新</el-button></div>
      <el-table :data="filtered" v-loading="loading" empty-text="当前命名空间还没有应用">
        <el-table-column label="应用"><template #default="{ row }"><div class="primary-cell"><span class="resource-avatar app-avatar">A</span><div><el-button link class="app-name-link" @click="$emit('view-deployments', row.name)">{{ row.name }}</el-button><small>{{ row.repo }}</small></div></div></template></el-table-column>
        <el-table-column label="规格" width="140"><template #default="{ row }">{{ row.cpu }} Core · {{ row.memory }} GB</template></el-table-column>
        <el-table-column label="触发规则" width="150"><template #default="{ row }"><el-tag effect="plain">{{ row.trigger?.type }} · {{ row.trigger?.rule }}</el-tag></template></el-table-column>
        <el-table-column label="访问地址"><template #default="{ row }"><span v-if="row.url?.length">{{ row.url[0].host }} : {{ row.url[0].port }}</span><span v-else class="muted">—</span></template></el-table-column>
        <el-table-column align="right" width="280"><template #default="{ row }"><el-button link @click="openEdit(row)">编辑</el-button><el-button link type="primary" @click="openDeploy(row)">部署</el-button><el-button link @click="$emit('view-deployments', row.name)">部署记录</el-button><el-button link type="danger" @click="remove(row)">删除</el-button></template></el-table-column>
      </el-table>
    </el-card>
  </section>

  <el-dialog v-model="dialog" :title="editing ? '编辑应用' : '创建应用'" width="660px" destroy-on-close>
    <el-form label-position="top" class="form-grid">
      <el-form-item label="应用名称" required><el-input v-model="form.name" placeholder="api-service" :disabled="editing" /></el-form-item>
      <el-form-item label="Git 仓库" required><el-input v-model="form.repo" placeholder="https://github.com/team/repo.git" /></el-form-item>
      <el-form-item label="访问令牌"><el-input v-model="form.token" type="password" show-password placeholder="可使用已配置的仓库凭据" /></el-form-item>
      <el-form-item label="触发方式"><el-select v-model="form.trigger.type"><el-option label="分支" value="branch"/><el-option label="标签" value="tag"/></el-select></el-form-item>
      <el-form-item label="触发规则" required><el-input v-model="form.trigger.rule" placeholder="main" /></el-form-item>
      <el-form-item label="CPU (Core)"><el-input-number v-model="form.cpu" :min="0.1" :step="0.5" /></el-form-item>
      <el-form-item label="内存 (GB)"><el-input-number v-model="form.memory" :min="1" /></el-form-item>
      <el-divider>环境变量</el-divider>
      <div class="env-list form-span">
        <div v-for="(env, index) in form.env" :key="index" class="env-row">
          <el-input v-model="env.key" placeholder="变量名，例如 NODE_ENV" />
          <el-input v-model="env.value" placeholder="变量值" show-password />
          <el-button aria-label="删除环境变量" @click="removeEnv(index)">删除</el-button>
        </div>
        <el-button plain class="env-add" @click="addEnv">添加环境变量</el-button>
      </div>
      <el-divider>访问地址</el-divider>
      <el-form-item label="域名 / 路径" required><el-input v-model="form.url[0].host" placeholder="app.example.com/api" /></el-form-item>
      <el-form-item label="容器端口" required><el-input v-model="form.url[0].port" placeholder="8080" /></el-form-item>
    </el-form>
    <template #footer><el-button @click="dialog=false">取消</el-button><el-button type="primary" :loading="saving" @click="save">{{ editing ? '保存修改' : '创建应用' }}</el-button></template>
  </el-dialog>

  <el-dialog v-model="deployDialog" :title="`部署 ${selected?.name || ''}`" width="480px">
    <el-form label-position="top"><el-form-item label="部署来源"><el-radio-group v-model="deployType"><el-radio-button value="branch">分支</el-radio-button><el-radio-button value="tag">标签</el-radio-button><el-radio-button value="commit">Commit</el-radio-button></el-radio-group></el-form-item><el-form-item :label="deployType"><el-input v-model="deployValue" :placeholder="deployType === 'branch' ? 'main' : ''" /></el-form-item></el-form>
    <template #footer><el-button @click="deployDialog=false">取消</el-button><el-button type="primary" :loading="deploying" @click="deploy">开始部署</el-button></template>
  </el-dialog>

  <el-dialog v-model="buildLogsDialog" :title="`${selected?.name || ''} 打包日志`" width="900px" @closed="stopPolling">
    <div class="log-toolbar">
      <el-select v-model="selectedJobId" placeholder="选择部署任务" @change="selectBuildJob">
        <el-option v-for="job in deploymentJobs" :key="job.id" :label="`${formatJobTime(job.startedAt)} · ${job.status}`" :value="job.id" />
      </el-select>
      <el-tag :type="jobTagType" effect="dark">{{ currentJob?.status || '暂无任务' }}</el-tag>
      <el-button :loading="buildLogsLoading" @click="refreshBuildJob">刷新</el-button>
    </div>
    <pre v-loading="buildLogsLoading && !currentJob" class="log-output build-log-output">{{ currentJob?.logs || '暂无打包日志' }}</pre>
    <el-alert v-if="currentJob?.error" :title="currentJob.error" type="error" :closable="false" show-icon class="deploy-error" />
  </el-dialog>

  <el-dialog v-model="logsDialog" :title="`${selected?.name || ''} 部署日志`" width="820px">
    <div class="log-toolbar">
      <el-select v-model="selectedContainer" @change="loadLogs">
        <el-option v-for="item in selected?.deploy || []" :key="item.containerId" :label="`${item.version} · ${item.containerId.slice(0, 12)}`" :value="item.containerId" />
      </el-select>
      <el-select v-model="logTail" class="tail-select" @change="loadLogs"><el-option label="最近 100 行" value="100"/><el-option label="最近 200 行" value="200"/><el-option label="最近 500 行" value="500"/><el-option label="最近 1000 行" value="1000"/></el-select>
      <el-button :loading="logsLoading" @click="loadLogs">刷新</el-button>
    </div>
    <pre v-loading="logsLoading" class="log-output">{{ logs || '暂无日志输出' }}</pre>
  </el-dialog>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, reactive, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { api, type AppRecord } from '../api'
const props = defineProps<{ namespace: string }>()
defineEmits<{ 'view-deployments': [app: string] }>()
type DeploymentJob={id:string;status:'running'|'succeeded'|'failed';logs:string;error?:string;startedAt:string;finishedAt?:string}
const records=ref<AppRecord[]>([]), loading=ref(false), search=ref(''), dialog=ref(false), editing=ref(false), saving=ref(false), deployDialog=ref(false), deploying=ref(false), selected=ref<AppRecord>(), deployType=ref('branch'), deployValue=ref('main'), logsDialog=ref(false), logsLoading=ref(false), selectedContainer=ref(''), logTail=ref('200'), logs=ref(''), buildLogsDialog=ref(false), buildLogsLoading=ref(false), deploymentJobs=ref<DeploymentJob[]>([]), selectedJobId=ref(''), currentJob=ref<DeploymentJob>()
let pollTimer:number|undefined
const jobTagType=computed(()=>currentJob.value?.status==='succeeded'?'success':currentJob.value?.status==='failed'?'danger':'warning')
const makeForm=()=>({name:'',repo:'',token:'',cpu:1,memory:1,trigger:{type:'branch' as 'branch'|'tag',rule:'main'},env:[] as Array<{key:string;value:string}>,url:[{host:'',port:'8080'}]})
const form=reactive(makeForm())
const filtered=computed(()=>records.value.filter(v=>`${v.name} ${v.repo}`.toLowerCase().includes(search.value.toLowerCase())))
async function load(){loading.value=true;try{records.value=await api.get<AppRecord[]>(`/namespaces/${props.namespace}/apps`)||[]}catch(e){ElMessage.error((e as Error).message)}finally{loading.value=false}}
function openCreate(){editing.value=false;Object.assign(form,makeForm());dialog.value=true}
function openEdit(row:AppRecord){editing.value=true;Object.assign(form,{...makeForm(),name:row.name,repo:row.repo,cpu:row.cpu,memory:row.memory,trigger:{...row.trigger},env:(row.env||[]).map(env=>({...env})),url:(row.url||[]).map(url=>({...url}))});dialog.value=true}
function addEnv(){form.env.push({key:'',value:''})}
function removeEnv(index:number){form.env.splice(index,1)}
async function save(){if(!form.name||!form.repo||!form.url[0]?.host||!form.url[0]?.port)return ElMessage.warning('请填写必填项');const invalidEnv=form.env.some(env=>!env.key.trim());if(invalidEnv)return ElMessage.warning('环境变量名称不能为空');saving.value=true;try{const data={...form,env:form.env.map(env=>({key:env.key.trim(),value:env.value}))};if(editing.value)await api.put(`/namespaces/${props.namespace}/apps/${form.name}`,data);else await api.post(`/namespaces/${props.namespace}/apps`,data);dialog.value=false;ElMessage.success(editing.value?'应用配置已更新':'应用已创建');load()}catch(e){ElMessage.error((e as Error).message)}finally{saving.value=false}}
function openDeploy(row:AppRecord){selected.value=row;deployType.value=row.trigger?.type||'branch';deployValue.value=row.trigger?.rule||'main';deployDialog.value=true}
async function deploy(){if(!deployValue.value)return ElMessage.warning('请输入部署来源');deploying.value=true;try{const job=await api.post<DeploymentJob>(`/namespaces/${props.namespace}/apps/${selected.value!.name}/deploy`,{[deployType.value]:deployValue.value});deployDialog.value=false;deploymentJobs.value=[job,...deploymentJobs.value.filter(item=>item.id!==job.id)];selectedJobId.value=job.id;currentJob.value=job;buildLogsDialog.value=true;startPolling();ElMessage.success('部署任务已进入后台')}catch(e){ElMessage.error((e as Error).message)}finally{deploying.value=false}}
function selectBuildJob(){currentJob.value=deploymentJobs.value.find(job=>job.id===selectedJobId.value);if(currentJob.value?.status==='running')startPolling();else stopPolling()}
async function refreshBuildJob(){if(!selected.value||!selectedJobId.value)return;buildLogsLoading.value=true;try{const job=await api.get<DeploymentJob>(`/namespaces/${props.namespace}/apps/${selected.value.name}/deploy/jobs/${selectedJobId.value}`);currentJob.value=job;const index=deploymentJobs.value.findIndex(item=>item.id===job.id);if(index>=0)deploymentJobs.value[index]=job;else deploymentJobs.value.unshift(job);if(job.status!=='running'){stopPolling();load()}}catch(e){stopPolling();ElMessage.error((e as Error).message)}finally{buildLogsLoading.value=false}}
function startPolling(){stopPolling();refreshBuildJob();pollTimer=window.setInterval(refreshBuildJob,1000)}
function stopPolling(){if(pollTimer!==undefined){window.clearInterval(pollTimer);pollTimer=undefined}}
function formatJobTime(value:string){return new Date(value).toLocaleString()}
async function loadLogs(){if(!selected.value||!selectedContainer.value)return;logsLoading.value=true;try{const result=await api.get<{logs:string}>(`/namespaces/${props.namespace}/apps/${selected.value.name}/deploy/${selectedContainer.value}/logs?tail=${logTail.value}`);logs.value=result.logs}catch(e){logs.value='';ElMessage.error((e as Error).message)}finally{logsLoading.value=false}}
async function remove(row:AppRecord){try{await ElMessageBox.confirm(`确定删除应用 “${row.name}” 及其容器吗？`,'删除应用',{type:'warning'});await api.delete(`/namespaces/${props.namespace}/apps/${row.name}`);ElMessage.success('应用已删除');load()}catch(e){if(e!=='cancel'&&e!=='close')ElMessage.error((e as Error).message)}}
watch(()=>props.namespace,load,{immediate:true})
onBeforeUnmount(stopPolling)
</script>

<style scoped>
.env-list { display: flex; flex-direction: column; gap: 10px; margin-bottom: 4px; }
.env-row { display: grid; grid-template-columns: 1fr 1fr auto; gap: 10px; }
.env-add { align-self: flex-start; }
.log-toolbar { display: flex; gap: 10px; margin-bottom: 12px; }
.log-toolbar .el-select:first-child { flex: 1; }
.tail-select { width: 150px; }
.log-output { min-height: 360px; max-height: 60vh; margin: 0; padding: 16px; overflow: auto; border-radius: 8px; background: #101521; color: #d8e0ee; font: 12px/1.6 ui-monospace, SFMono-Regular, Consolas, monospace; white-space: pre-wrap; word-break: break-all; }
.build-log-output { min-height: 440px; }
.deploy-error { margin-top: 12px; }
.app-name-link { height: auto; padding: 0; color: #1b273d; font-weight: 600; }
</style>
