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
        <el-table-column align="right" width="230"><template #default="{ row }"><el-button link @click="openEdit(row)">编辑</el-button><el-button link type="primary" @click="$emit('view-deployments', row.name)">部署记录</el-button><el-button link type="danger" @click="remove(row)">删除</el-button></template></el-table-column>
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
      <el-form-item label="自动重启策略" class="form-span"><el-select v-model="form.restart_policy"><el-option label="除非手动停止（推荐）" value="unless-stopped"/><el-option label="始终自动重启" value="always"/><el-option label="仅失败时重启" value="on-failure"/><el-option label="不自动重启" value="no"/></el-select><div class="form-hint">应用下次部署的新容器将使用该策略。</div></el-form-item>
      <el-divider>环境变量</el-divider>
      <div class="env-list form-span">
        <div v-for="(env, index) in form.env" :key="index" class="env-row">
          <el-input v-model="env.key" placeholder="变量名，例如 NODE_ENV" />
          <el-input v-model="env.value" placeholder="变量值" show-password />
          <el-button aria-label="删除环境变量" @click="removeEnv(index)">删除</el-button>
        </div>
        <div class="env-actions"><el-button plain @click="addEnv">添加环境变量</el-button><el-button plain @click="openBatchEnv">批量输入</el-button></div>
      </div>
      <el-divider>访问地址</el-divider>
      <el-form-item label="域名 / 路径" required><el-input v-model="form.url[0].host" placeholder="app.example.com/api" /></el-form-item>
      <el-form-item label="容器端口" required><el-input v-model="form.url[0].port" placeholder="8080" /></el-form-item>
    </el-form>
    <template #footer><el-button @click="dialog=false">取消</el-button><el-button type="primary" :loading="saving" @click="save">{{ editing ? '保存修改' : '创建应用' }}</el-button></template>
  </el-dialog>

  <el-dialog v-model="batchEnvDialog" title="批量输入环境变量" width="720px" append-to-body>
    <p class="batch-env-hint">每行一个变量，格式为 <code>KEY=VALUE</code>。支持空行和以 # 开头的注释。</p>
    <el-input v-model="batchEnvText" type="textarea" :rows="16" resize="vertical" class="batch-env-input" placeholder="NODE_ENV=production&#10;API_URL=https://api.example.com&#10;TOKEN=value=with=equals" />
    <template #footer><el-button @click="batchEnvDialog=false">取消</el-button><el-button type="primary" @click="applyBatchEnv">应用</el-button></template>
  </el-dialog>

</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { api, type AppRecord } from '../api'
const props = defineProps<{ namespace: string }>()
defineEmits<{ 'view-deployments': [app: string] }>()
const records=ref<AppRecord[]>([]), loading=ref(false), search=ref(''), dialog=ref(false), editing=ref(false), saving=ref(false)
const batchEnvDialog=ref(false), batchEnvText=ref('')
const makeForm=()=>({name:'',repo:'',token:'',cpu:1,memory:1,restart_policy:'unless-stopped' as 'unless-stopped'|'always'|'on-failure'|'no',trigger:{type:'branch' as 'branch'|'tag',rule:'main'},env:[] as Array<{key:string;value:string}>,url:[{host:'',port:'8080'}]})
const form=reactive(makeForm())
const filtered=computed(()=>records.value.filter(v=>`${v.name} ${v.repo}`.toLowerCase().includes(search.value.toLowerCase())))
async function load(){loading.value=true;try{records.value=await api.get<AppRecord[]>(`/namespaces/${props.namespace}/apps`)||[]}catch(e){ElMessage.error((e as Error).message)}finally{loading.value=false}}
function openCreate(){editing.value=false;Object.assign(form,makeForm());dialog.value=true}
function openEdit(row:AppRecord){editing.value=true;Object.assign(form,{...makeForm(),name:row.name,repo:row.repo,cpu:row.cpu,memory:row.memory,restart_policy:row.restart_policy||'unless-stopped',trigger:{...row.trigger},env:(row.env||[]).map(env=>({...env})),url:(row.url||[]).map(url=>({...url}))});dialog.value=true}
function addEnv(){form.env.push({key:'',value:''})}
function removeEnv(index:number){form.env.splice(index,1)}
function openBatchEnv(){batchEnvText.value=form.env.filter(env=>env.key.trim()).map(env=>`${env.key}=${env.value}`).join('\n');batchEnvDialog.value=true}
function applyBatchEnv(){const parsed=new Map<string,string>();const lines=batchEnvText.value.split(/\r?\n/);for(let index=0;index<lines.length;index++){let line=lines[index].trim();if(!line||line.startsWith('#'))continue;if(line.startsWith('export '))line=line.slice(7).trim();const separator=line.indexOf('=');if(separator<1){ElMessage.warning(`第 ${index+1} 行格式错误，应为 KEY=VALUE`);return}const key=line.slice(0,separator).trim();if(!/^[A-Za-z_][A-Za-z0-9_]*$/.test(key)){ElMessage.warning(`第 ${index+1} 行变量名无效：${key}`);return}parsed.set(key,line.slice(separator+1))}form.env=Array.from(parsed,([key,value])=>({key,value}));batchEnvDialog.value=false;ElMessage.success(`已应用 ${form.env.length} 个环境变量`)}
async function save(){if(!form.name||!form.repo||!form.url[0]?.host||!form.url[0]?.port)return ElMessage.warning('请填写必填项');const invalidEnv=form.env.some(env=>!env.key.trim());if(invalidEnv)return ElMessage.warning('环境变量名称不能为空');saving.value=true;try{const data={...form,env:form.env.map(env=>({key:env.key.trim(),value:env.value}))};if(editing.value)await api.put(`/namespaces/${props.namespace}/apps/${form.name}`,data);else await api.post(`/namespaces/${props.namespace}/apps`,data);dialog.value=false;ElMessage.success(editing.value?'应用配置已更新':'应用已创建');load()}catch(e){ElMessage.error((e as Error).message)}finally{saving.value=false}}
async function remove(row:AppRecord){try{await ElMessageBox.confirm(`确定删除应用 “${row.name}” 及其容器吗？`,'删除应用',{type:'warning'});await api.delete(`/namespaces/${props.namespace}/apps/${row.name}`);ElMessage.success('应用已删除');load()}catch(e){if(e!=='cancel'&&e!=='close')ElMessage.error((e as Error).message)}}
watch(()=>props.namespace,load,{immediate:true})
</script>

<style scoped>
.env-list { display: flex; flex-direction: column; gap: 10px; margin-bottom: 4px; }
.env-row { display: grid; grid-template-columns: 1fr 1fr auto; gap: 10px; }
.env-actions { display: flex; gap: 8px; align-self: flex-start; }
.batch-env-hint { margin: 0 0 12px; color: #778195; font-size: 13px; }
.batch-env-hint code { color: #315fb8; }
.batch-env-input :deep(textarea) { font-family: ui-monospace, SFMono-Regular, Consolas, monospace; line-height: 1.6; }
.app-name-link { height: auto; padding: 0; color: #1b273d; font-weight: 600; }
</style>
