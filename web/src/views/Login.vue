<template>
  <main class="login-page">
    <section class="login-panel">
      <div class="login-brand"><span class="brand-mark">D</span><span>DockFlow</span></div>
      <div class="login-copy"><p class="eyebrow">SERVER CONSOLE</p><h1>登录管理面板</h1><p>使用这台 Linux 服务器的系统账号和密码登录。</p></div>
      <el-form label-position="top" @submit.prevent="submit">
        <el-form-item label="Linux 用户名"><el-input v-model="username" size="large" autocomplete="username" autofocus placeholder="请输入系统用户名" @keyup.enter="submit" /></el-form-item>
        <el-form-item label="密码"><el-input v-model="password" type="password" size="large" autocomplete="current-password" show-password placeholder="请输入系统密码" @keyup.enter="submit" /></el-form-item>
        <el-button class="login-submit" type="primary" size="large" native-type="submit" :loading="loading" @click="submit">登录</el-button>
      </el-form>
      <p class="login-note">凭据仅用于 Linux PAM 校验，不会保存在 DockFlow 中。</p>
    </section>
    <aside class="login-visual"><div class="visual-grid"></div><div class="visual-content"><span class="visual-kicker">DOCKFLOW</span><h2>让部署保持<br>简单、清晰、可控。</h2><p>在一个界面中管理应用、数据库、Redis 和代码仓库。</p></div></aside>
  </main>
</template>
<script setup lang="ts">
import { ref } from 'vue'
import { ElMessage } from 'element-plus'
import { useRoute, useRouter } from 'vue-router'
import { api } from '../api'
import { auth } from '../auth'
const route = useRoute(); const router = useRouter(); const username = ref(''); const password = ref(''); const loading = ref(false)
async function submit() {
  if (loading.value) return
  if (!username.value.trim() || !password.value) return ElMessage.warning('请输入用户名和密码')
  loading.value = true
  try {
    const value = await api.post<{ username: string }>('/auth/login', { username: username.value.trim(), password: password.value })
    auth.username = value.username; password.value = ''
    const redirect = typeof route.query.redirect === 'string' && route.query.redirect.startsWith('/') ? route.query.redirect : '/apps'
    await router.replace(redirect)
  } catch (error) { ElMessage.error((error as Error).message) } finally { loading.value = false }
}
</script>
