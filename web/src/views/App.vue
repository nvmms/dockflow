<template>
  <section class="resource-page">
    <header class="page-header"><div><p class="eyebrow">WORKLOADS</p><h1>应用</h1><p class="page-description">从 Git 仓库构建、部署和管理服务。</p></div><el-button type="primary" @click="openCreate">创建应用</el-button></header>
    <el-card shadow="never" class="resource-card">
      <div class="table-toolbar"><el-input v-model="search" placeholder="搜索应用或仓库" clearable class="search-input" /><el-button text :loading="loading" @click="load">刷新</el-button></div>
      <el-table :data="filtered" v-loading="loading" empty-text="当前命名空间还没有应用">
        <el-table-column label="应用"><template #default="{ row }"><div class="primary-cell"><span class="resource-avatar app-avatar">A</span><div><strong>{{ row.name }}</strong><small>{{ row.repo }}</small></div></div></template></el-table-column>
        <el-table-column label="规格" width="140"><template #default="{ row }">{{ row.cpu }} Core · {{ row.memory }} GB</template></el-table-column>
        <el-table-column label="触发规则" width="150"><template #default="{ row }"><el-tag effect="plain">{{ row.trigger?.type }} · {{ row.trigger?.rule }}</el-tag></template></el-table-column>
        <el-table-column label="访问地址"><template #default="{ row }"><span v-if="row.url?.length">{{ row.url[0].host }} : {{ row.url[0].port }}</span><span v-else class="muted">—</span></template></el-table-column>
        <el-table-column align="right" width="185"><template #default="{ row }"><el-button link type="primary" @click="openDeploy(row)">部署</el-button><el-button link type="danger" @click="remove(row)">删除</el-button></template></el-table-column>
      </el-table>
    </el-card>
  </section>

  <el-dialog v-model="dialog" title="创建应用" width="660px" destroy-on-close>
    <el-form label-position="top" class="form-grid">
      <el-form-item label="应用名称" required><el-input v-model="form.name" placeholder="api-service" /></el-form-item>
      <el-form-item label="Git 仓库" required><el-input v-model="form.repo" placeholder="https://github.com/team/repo.git" /></el-form-item>
      <el-form-item label="访问令牌"><el-input v-model="form.token" type="password" show-password placeholder="可使用已配置的仓库凭据" /></el-form-item>
      <el-form-item label="触发方式"><el-select v-model="form.trigger.type"><el-option label="分支" value="branch"/><el-option label="标签" value="tag"/></el-select></el-form-item>
      <el-form-item label="触发规则" required><el-input v-model="form.trigger.rule" placeholder="main" /></el-form-item>
      <el-form-item label="CPU (Core)"><el-input-number v-model="form.cpu" :min="0.1" :step="0.5" /></el-form-item>
      <el-form-item label="内存 (GB)"><el-input-number v-model="form.memory" :min="1" /></el-form-item>
      <el-divider>访问地址</el-divider>
      <el-form-item label="域名 / 路径" required><el-input v-model="form.url[0].host" placeholder="app.example.com/api" /></el-form-item>
      <el-form-item label="容器端口" required><el-input v-model="form.url[0].port" placeholder="8080" /></el-form-item>
    </el-form>
    <template #footer><el-button @click="dialog=false">取消</el-button><el-button type="primary" :loading="saving" @click="create">创建应用</el-button></template>
  </el-dialog>

  <el-dialog v-model="deployDialog" :title="`部署 ${selected?.name || ''}`" width="480px">
    <el-form label-position="top"><el-form-item label="部署来源"><el-radio-group v-model="deployType"><el-radio-button value="branch">分支</el-radio-button><el-radio-button value="tag">标签</el-radio-button><el-radio-button value="commit">Commit</el-radio-button></el-radio-group></el-form-item><el-form-item :label="deployType"><el-input v-model="deployValue" :placeholder="deployType === 'branch' ? 'main' : ''" /></el-form-item></el-form>
    <template #footer><el-button @click="deployDialog=false">取消</el-button><el-button type="primary" :loading="deploying" @click="deploy">开始部署</el-button></template>
  </el-dialog>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { api, type AppRecord } from '../api'
const props = defineProps<{ namespace: string }>()
const records=ref<AppRecord[]>([]), loading=ref(false), search=ref(''), dialog=ref(false), saving=ref(false), deployDialog=ref(false), deploying=ref(false), selected=ref<AppRecord>(), deployType=ref('branch'), deployValue=ref('main')
const makeForm=()=>({name:'',repo:'',token:'',cpu:1,memory:1,trigger:{type:'branch' as 'branch'|'tag',rule:'main'},env:[],url:[{host:'',port:'8080'}]})
const form=reactive(makeForm())
const filtered=computed(()=>records.value.filter(v=>`${v.name} ${v.repo}`.toLowerCase().includes(search.value.toLowerCase())))
async function load(){loading.value=true;try{records.value=await api.get<AppRecord[]>(`/namespaces/${props.namespace}/apps`)||[]}catch(e){ElMessage.error((e as Error).message)}finally{loading.value=false}}
function openCreate(){Object.assign(form,makeForm());dialog.value=true}
async function create(){if(!form.name||!form.repo||!form.url[0].host||!form.url[0].port)return ElMessage.warning('请填写必填项');saving.value=true;try{await api.post(`/namespaces/${props.namespace}/apps`,form);dialog.value=false;ElMessage.success('应用已创建');load()}catch(e){ElMessage.error((e as Error).message)}finally{saving.value=false}}
function openDeploy(row:AppRecord){selected.value=row;deployType.value=row.trigger?.type||'branch';deployValue.value=row.trigger?.rule||'main';deployDialog.value=true}
async function deploy(){if(!deployValue.value)return ElMessage.warning('请输入部署来源');deploying.value=true;try{await api.post(`/namespaces/${props.namespace}/apps/${selected.value!.name}/deploy`,{[deployType.value]:deployValue.value});deployDialog.value=false;ElMessage.success('部署完成');load()}catch(e){ElMessage.error((e as Error).message)}finally{deploying.value=false}}
async function remove(row:AppRecord){try{await ElMessageBox.confirm(`确定删除应用 “${row.name}” 及其容器吗？`,'删除应用',{type:'warning'});await api.delete(`/namespaces/${props.namespace}/apps/${row.name}`);ElMessage.success('应用已删除');load()}catch(e){if(e!=='cancel'&&e!=='close')ElMessage.error((e as Error).message)}}
watch(()=>props.namespace,load,{immediate:true})
</script>
