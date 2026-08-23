<template>
  <section class="resource-page">
    <header class="page-header"><div><p class="eyebrow">DATA SERVICES</p><h1>数据库</h1><p class="page-description">运行隔离的 MySQL 或 PostgreSQL 数据库。</p></div><el-button type="primary" @click="dialog=true">创建数据库</el-button></header>
    <el-card shadow="never" class="resource-card"><div class="table-toolbar"><span class="record-count">{{ records.length }} 个实例</span><el-button text :loading="loading" @click="load">刷新</el-button></div>
      <el-table :data="records" v-loading="loading" empty-text="当前命名空间还没有数据库">
        <el-table-column label="数据库"><template #default="{row}"><div class="primary-cell"><span class="resource-avatar db-avatar">DB</span><div><strong>{{row.name}}</strong><small>{{row.dbname}}</small></div></div></template></el-table-column>
        <el-table-column label="引擎" width="150"><template #default="{row}"><el-tag effect="plain">{{row.dbtype}}</el-tag></template></el-table-column>
        <el-table-column label="规格" width="140"><template #default="{row}">{{row.cpu}} Core · {{row.memory}} GB</template></el-table-column>
        <el-table-column label="状态" width="120"><template #default="{row}"><el-tag :type="statusTag(row.status)" effect="plain">{{statusText(row.status)}}</el-tag></template></el-table-column>
        <el-table-column label="内网地址"><template #default="{row}">{{row.ip?.join(', ')||'—'}}</template></el-table-column>
        <el-table-column align="right" width="330"><template #default="{row}"><el-button v-if="row.status==='running'" link :loading="operating===row.name" @click="setRunning(row,false)">停止</el-button><el-button v-if="row.status==='stopped'" link type="primary" :loading="operating===row.name" @click="setRunning(row,true)">启动</el-button><el-button link :disabled="row.status!=='running'" @click="exportSQL(row)">导出</el-button><el-button link :disabled="row.status!=='running'" @click="chooseImport(row)">导入</el-button><el-button link type="danger" :disabled="row.status==='importing'" @click="remove(row)">删除</el-button></template></el-table-column>
      </el-table><input ref="fileInput" hidden type="file" accept=".sql,application/sql" @change="importSQL" />
    </el-card>
  </section>
  <el-dialog v-model="dialog" title="创建数据库" width="620px" destroy-on-close><el-form label-position="top" class="form-grid">
    <el-form-item label="实例名称" required><el-input v-model="form.name" placeholder="main-db"/></el-form-item><el-form-item label="数据库引擎"><el-select v-model="form.dbtype"><el-option label="MySQL 5.7" value="mysql:5.7"/><el-option label="MySQL 8.0" value="mysql:8.0"/><el-option label="PostgreSQL 16" value="postgres:16"/></el-select></el-form-item>
    <el-form-item label="数据库名" required><el-input v-model="form.dbname"/></el-form-item><el-form-item label="用户名" required><el-input v-model="form.username"/></el-form-item><el-form-item label="密码" required><el-input v-model="form.password" type="password" show-password/></el-form-item><el-form-item label="允许远程访问"><el-switch v-model="form.remote"/></el-form-item><el-form-item label="CPU (Core)"><el-input-number v-model="form.cpu" :min="0.5" :step="0.5"/></el-form-item><el-form-item label="内存 (GB)"><el-input-number v-model="form.memory" :min="0.5" :step="0.5"/></el-form-item>
  </el-form><template #footer><el-button @click="dialog=false">取消</el-button><el-button type="primary" :loading="saving" @click="create">创建数据库</el-button></template></el-dialog>
</template>
<script setup lang="ts">
import { onBeforeUnmount, reactive, ref, watch } from 'vue';import { ElMessage,ElMessageBox } from 'element-plus';import { api,type DatabaseRecord } from '../api'
const props=defineProps<{namespace:string}>();const records=ref<DatabaseRecord[]>([]),loading=ref(false),dialog=ref(false),saving=ref(false),fileInput=ref<HTMLInputElement>(),importTarget=ref<DatabaseRecord>(),operating=ref('');const form=reactive({name:'',dbtype:'mysql:5.7',dbname:'',username:'',password:'',remote:false,cpu:1,memory:2})
let pollTimer:number|undefined
const reportedImportErrors=new Set<string>()
async function load(){loading.value=true;try{records.value=await api.get<DatabaseRecord[]>(`/namespaces/${props.namespace}/databases`)||[];for(const row of records.value){if(row.import_error){const key=`${row.name}:${row.import_error}`;if(!reportedImportErrors.has(key)){reportedImportErrors.add(key);ElMessage.error(`${row.name} 导入失败：${row.import_error}`)}}}updatePolling()}catch(e){ElMessage.error((e as Error).message)}finally{loading.value=false}}
function updatePolling(){const importing=records.value.some(row=>row.status==='importing');if(importing&&pollTimer===undefined)pollTimer=window.setInterval(load,2000);if(!importing&&pollTimer!==undefined){window.clearInterval(pollTimer);pollTimer=undefined}}
function statusText(status?:string){return status==='running'?'运行中':status==='stopped'?'已停止':status==='missing'?'容器不存在':status==='paused'?'已暂停':status==='restarting'?'重启中':status==='importing'?'导入中':'未知'}
function statusTag(status?:string){return status==='running'?'success':status==='importing'||status==='restarting'?'warning':status==='missing'?'danger':'info'}
async function setRunning(row:DatabaseRecord,running:boolean){operating.value=row.name;try{await api.post(`/namespaces/${props.namespace}/databases/${row.name}/${running?'start':'stop'}`);ElMessage.success(running?'数据库已启动':'数据库已停止');await load()}catch(e){ElMessage.error((e as Error).message)}finally{operating.value=''}}
async function create(){if(!form.name||!form.dbname||!form.username||!form.password)return ElMessage.warning('请填写必填项');saving.value=true;try{await api.post(`/namespaces/${props.namespace}/databases`,form);dialog.value=false;ElMessage.success('数据库已创建');load()}catch(e){ElMessage.error((e as Error).message)}finally{saving.value=false}}
function exportSQL(row:DatabaseRecord){window.open(`/api/v1/namespaces/${props.namespace}/databases/${row.name}/export`,'_blank')}
function chooseImport(row:DatabaseRecord){importTarget.value=row;fileInput.value?.click()}
async function importSQL(event:Event){const input=event.target as HTMLInputElement,file=input.files?.[0];if(!file||!importTarget.value)return;try{const res=await fetch(`/api/v1/namespaces/${props.namespace}/databases/${importTarget.value.name}/import`,{method:'POST',headers:{'Content-Type':'application/sql'},body:file});if(!res.ok)throw new Error((await res.json()).error);ElMessage.success('SQL 上传完成，正在后台导入');await load()}catch(e){ElMessage.error((e as Error).message)}finally{input.value=''}}
async function remove(row:DatabaseRecord){try{await ElMessageBox.confirm(`确定删除数据库 “${row.name}” 及其数据卷吗？`,'删除数据库',{type:'warning'});await api.delete(`/namespaces/${props.namespace}/databases/${row.name}`);ElMessage.success('数据库已删除');load()}catch(e){if(e!=='cancel'&&e!=='close')ElMessage.error((e as Error).message)}}watch(()=>props.namespace,load,{immediate:true})
onBeforeUnmount(()=>{if(pollTimer!==undefined)window.clearInterval(pollTimer)})
</script>
