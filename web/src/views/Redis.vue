<template>
  <section class="resource-page">
    <header class="page-header">
      <div><p class="eyebrow">DATA SERVICES</p><h1>Redis</h1><p class="page-description">创建低延迟缓存和键值存储实例。</p></div>
      <el-button type="primary" @click="dialog=true">创建 Redis</el-button>
    </header>
    <el-card shadow="never" class="resource-card">
      <div class="table-toolbar"><span class="record-count">{{records.length}} 个实例</span><el-button text :loading="loading" @click="load">刷新</el-button></div>
      <el-table :data="records" v-loading="loading" empty-text="当前命名空间还没有 Redis 实例">
        <el-table-column label="实例"><template #default="{row}"><div class="primary-cell"><span class="resource-avatar redis-avatar">R</span><div><strong>{{row.name}}</strong><small>Redis {{row.version}}</small></div></div></template></el-table-column>
        <el-table-column label="状态" width="120"><template #default="{row}"><el-tag :type="statusTag(row.status)" effect="plain">{{statusText(row.status)}}</el-tag></template></el-table-column>
        <el-table-column label="配置" width="170"><template #default="{row}"><span>{{restartPolicyText(row.restart_policy)}}</span><el-tag v-if="row.needs_recreate" type="warning" size="small" effect="plain" class="config-pending">需重建</el-tag></template></el-table-column>
        <el-table-column label="持久化" width="140"><template #default="{row}"><el-tag :type="row.appendonly?'success':'info'" effect="plain">AOF {{row.appendonly?'已开启':'已关闭'}}</el-tag></template></el-table-column>
        <el-table-column label="规格" width="140"><template #default="{row}">{{row.cpu}} Core · {{row.memory}} GB</template></el-table-column>
        <el-table-column label="淘汰策略" prop="maxmemory_policy"/>
        <el-table-column label="内网地址"><template #default="{row}">{{row.ip?.join(', ')||'—'}}</template></el-table-column>
        <el-table-column align="right" width="235"><template #default="{row}"><el-button link @click="openEdit(row)">编辑</el-button><el-button v-if="row.status==='running'" link :loading="operating===row.name" @click="setRunning(row,false)">停止</el-button><el-button v-if="row.status==='stopped'" link type="primary" :loading="operating===row.name" @click="setRunning(row,true)">启动</el-button><el-button link type="danger" @click="remove(row)">删除</el-button></template></el-table-column>
      </el-table>
    </el-card>
  </section>
  <el-dialog v-model="editDialog" :title="`编辑 Redis ${editTarget?.name || ''}`" width="520px"><el-form label-position="top"><el-form-item label="自动重启策略"><el-select v-model="editRestartPolicy"><el-option label="除非手动停止（推荐）" value="unless-stopped"/><el-option label="始终自动重启" value="always"/><el-option label="仅失败时重启" value="on-failure"/><el-option label="不自动重启" value="no"/></el-select></el-form-item><el-divider>日志策略</el-divider><el-form-item label="日志驱动"><el-select v-model="editLogDriver"><el-option label="local（推荐）" value="local"/><el-option label="json-file" value="json-file"/><el-option label="阿里云日志服务" value="aliyun-sls"/></el-select></el-form-item><div v-if="editLogDriver!=='aliyun-sls'" class="form-grid"><el-form-item label="单文件上限"><el-input v-model="editLogMaxSize" placeholder="10m"/></el-form-item><el-form-item label="保留文件数"><el-input-number v-model="editLogMaxFile" :min="1" :max="100"/></el-form-item></div><div v-else class="form-grid"><el-form-item label="Endpoint"><el-input v-model="editSLSEndpoint"/></el-form-item><el-form-item label="Project"><el-input v-model="editSLSProject"/></el-form-item><el-form-item label="Logstore"><el-input v-model="editSLSLogstore"/></el-form-item><el-form-item label="采集配置名称"><el-input v-model="editSLSConfigName"/></el-form-item></div><el-form-item label="立即应用"><el-switch v-model="editApplyNow"/><div class="form-hint">只有选择阿里云日志服务的容器会添加 SLS 白名单 Label。旧 Redis 若没有持久化卷会拒绝重建。</div></el-form-item></el-form><template #footer><el-button @click="editDialog=false">取消</el-button><el-button type="primary" :loading="editSaving" @click="saveEdit">保存</el-button></template></el-dialog>
  <el-dialog v-model="dialog" title="创建 Redis" width="600px" destroy-on-close>
    <el-form label-position="top" class="form-grid">
      <el-form-item label="实例名称" required><el-input v-model="form.name" placeholder="session-cache"/></el-form-item>
      <el-form-item label="版本"><el-select v-model="form.version"><el-option label="Redis 7" value="7"/><el-option label="Redis 8" value="8"/></el-select></el-form-item>
      <el-form-item label="访问密码"><el-input v-model="form.password" type="password" show-password/></el-form-item>
      <el-form-item label="AOF 持久化"><el-switch v-model="form.appendonly"/></el-form-item>
      <el-form-item label="CPU (Core)"><el-input-number v-model="form.cpu" :min="0.1" :step="0.5"/></el-form-item>
      <el-form-item label="内存 (GB)"><el-input-number v-model="form.memory" :min="0.1" :step="0.5"/></el-form-item>
      <el-form-item label="内存淘汰策略" class="form-span"><el-select v-model="form.maxmemory_policy"><el-option label="allkeys-lru" value="allkeys-lru"/><el-option label="allkeys-lfu" value="allkeys-lfu"/><el-option label="volatile-lru" value="volatile-lru"/><el-option label="noeviction" value="noeviction"/></el-select></el-form-item>
      <el-form-item label="自动重启策略" class="form-span"><el-select v-model="form.restart_policy"><el-option label="除非手动停止（推荐）" value="unless-stopped"/><el-option label="始终自动重启" value="always"/><el-option label="仅失败时重启" value="on-failure"/><el-option label="不自动重启" value="no"/></el-select><div class="form-hint">控制容器异常退出或服务器重启后的恢复行为。</div></el-form-item>
    </el-form>
    <template #footer><el-button @click="dialog=false">取消</el-button><el-button type="primary" :loading="saving" @click="create">创建 Redis</el-button></template>
  </el-dialog>
</template>

<script setup lang="ts">
import { reactive, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { api, type RedisRecord } from '../api'

const props = defineProps<{namespace:string}>()
const records = ref<RedisRecord[]>([])
const loading = ref(false)
const dialog = ref(false)
const saving = ref(false)
const operating = ref('')
const editDialog = ref(false)
const editSaving = ref(false)
const editTarget = ref<RedisRecord>()
const editRestartPolicy = ref('unless-stopped')
const editLogDriver = ref('local')
const editLogMaxSize = ref('10m')
const editLogMaxFile = ref(3)
const editApplyNow = ref(false)
const editSLSProject = ref('')
const editSLSLogstore = ref('')
const editSLSEndpoint = ref('')
const editSLSConfigName = ref('')
const form = reactive({name:'',version:'7',password:'',appendonly:true,cpu:.5,memory:.5,maxmemory_policy:'allkeys-lru',restart_policy:'unless-stopped'})

async function load(){loading.value=true;try{records.value=await api.get<RedisRecord[]>(`/namespaces/${props.namespace}/redis`)||[]}catch(e){ElMessage.error((e as Error).message)}finally{loading.value=false}}
function statusText(status?:string){return status==='running'?'运行中':status==='stopped'?'已停止':status==='missing'?'容器不存在':status==='paused'?'已暂停':status==='restarting'?'重启中':'未知'}
function statusTag(status?:string){return status==='running'?'success':status==='restarting'?'warning':status==='missing'?'danger':'info'}
function restartPolicyText(value?:string){return value==='always'?'始终重启':value==='on-failure'?'失败时重启':value==='no'?'不自动重启':'除非手动停止'}
function openEdit(row:RedisRecord){editTarget.value=row;editRestartPolicy.value=row.restart_policy||'unless-stopped';editLogDriver.value=row.log_driver||'local';editLogMaxSize.value=row.log_max_size||'10m';editLogMaxFile.value=row.log_max_file||3;editSLSProject.value=row.sls_project||'';editSLSLogstore.value=row.sls_logstore||'';editSLSEndpoint.value=row.sls_endpoint||'';editSLSConfigName.value=row.sls_config_name||'';editApplyNow.value=false;editDialog.value=true}
async function saveEdit(){if(!editTarget.value)return;editSaving.value=true;try{await api.put(`/namespaces/${props.namespace}/redis/${editTarget.value.name}`,{restart_policy:editRestartPolicy.value,log_driver:editLogDriver.value,log_max_size:editLogMaxSize.value,log_max_file:editLogMaxFile.value,apply_now:editApplyNow.value,sls_project:editSLSProject.value,sls_logstore:editSLSLogstore.value,sls_endpoint:editSLSEndpoint.value,sls_config_name:editSLSConfigName.value});editDialog.value=false;ElMessage.success(editApplyNow.value?'Redis 已按新配置重建':'配置已保存，等待重建');await load()}catch(e){ElMessage.error((e as Error).message)}finally{editSaving.value=false}}
async function setRunning(row:RedisRecord,running:boolean){operating.value=row.name;try{await api.post(`/namespaces/${props.namespace}/redis/${row.name}/${running?'start':'stop'}`);ElMessage.success(running?'Redis 已启动':'Redis 已停止');await load()}catch(e){ElMessage.error((e as Error).message)}finally{operating.value=''}}
async function create(){if(!form.name)return ElMessage.warning('请输入实例名称');saving.value=true;try{await api.post(`/namespaces/${props.namespace}/redis`,form);dialog.value=false;ElMessage.success('Redis 已创建');load()}catch(e){ElMessage.error((e as Error).message)}finally{saving.value=false}}
async function remove(row:RedisRecord){try{await ElMessageBox.confirm(`确定删除 Redis “${row.name}” 吗？`,'删除 Redis',{type:'warning'});await api.delete(`/namespaces/${props.namespace}/redis/${row.name}`);ElMessage.success('Redis 已删除');load()}catch(e){if(e!=='cancel'&&e!=='close')ElMessage.error((e as Error).message)}}
watch(()=>props.namespace,load,{immediate:true})
</script>
