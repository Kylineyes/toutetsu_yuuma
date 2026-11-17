<script setup>
import { ref, onMounted } from 'vue'
import { api } from '../api.js'

const token = ref('')
const msg = ref('')
onMounted(async () => {
  try { const s = await api.getSettings(); token.value = s.api_token || '' } catch (e) { msg.value = e.message }
})
async function save() {
  msg.value = ''
  try {
    await api.saveSettings(token.value.trim())
    localStorage.setItem('api_token', token.value.trim())
    msg.value = '已保存（明文存 DB）'
  } catch (e) { msg.value = e.message }
}
</script>

<template>
  <h2>设置</h2>
  <div class="card">
    <p style="color:#64748b">API Token 明文存放在数据库 settings 表（不通过环境变量）。保存后本页会写入浏览器 localStorage，后续请求自动带上。设为空表示不启用鉴权。</p>
    <div class="row">
      <label class="fld" style="flex:1">API Token<input v-model="token" style="width:100%" placeholder="留空则不启用鉴权" /></label>
      <button @click="save">保存</button>
    </div>
    <p v-if="msg" style="color:#166534">{{ msg }}</p>
  </div>
</template>
