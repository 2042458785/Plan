<script setup lang="ts">
import { reactive, ref, watch } from 'vue'
import type { CustomerAccount } from '@/types/catalog'

const props = defineProps<{ account: CustomerAccount | null; secure: boolean; loading: boolean; error: string }>()
const emit = defineEmits<{
  authenticate: [register: boolean, input: { name: string; contact: string; password: string }]
  save: [input: { name: string; dorm: string }]
  logout: []
}>()
const registering = ref(false)
const input = reactive({ name: '', contact: '', password: '' })
const profile = reactive({ name: '', dorm: '' })
watch(() => props.account, account => {
  profile.name = account?.name ?? ''
  profile.dorm = account?.dorm ?? ''
  if (account) {
    input.password = ''
    input.contact = ''
    input.name = ''
  }
}, { immediate: true })
</script>

<template>
  <section class="account-panel" aria-label="顾客账号">
    <template v-if="account">
      <div class="account-identity">
        <div><span>顾客编号</span><strong>{{ account.number }}</strong></div>
        <button class="s-outline" type="button" :disabled="loading" @click="emit('logout')">退出登录</button>
      </div>
      <form class="account-form" @submit.prevent="emit('save', { ...profile })">
        <label>称呼<input v-model="profile.name" required maxlength="30" autocomplete="name"></label>
        <label>登录账号<input :value="account.contact" readonly></label>
        <label>默认配送地址 <small>选填</small><input v-model="profile.dorm" maxlength="100" placeholder="例如 3 栋 508 寝室" autocomplete="street-address"></label>
        <button class="s-outline" :disabled="loading" type="submit">保存资料</button>
      </form>
    </template>
    <template v-else-if="secure">
      <div class="account-tabs" aria-label="账号操作">
        <button type="button" :aria-pressed="!registering" @click="registering = false; input.password = ''">登录</button>
        <button type="button" :aria-pressed="registering" @click="registering = true; input.password = ''">注册</button>
      </div>
      <form class="account-form" @submit.prevent="emit('authenticate', registering, { ...input })">
        <label v-if="registering">称呼<input v-model="input.name" required maxlength="30" autocomplete="name"></label>
        <label>手机号或微信号<input v-model="input.contact" required maxlength="32" autocomplete="username" autocapitalize="none" spellcheck="false"></label>
        <label>密码<input v-model="input.password" required minlength="10" maxlength="72" type="password" :autocomplete="registering ? 'new-password' : 'current-password'"></label>
        <button class="s-primary" :disabled="loading" type="submit">{{ registering ? '注册并获取顾客编号' : '登录顾客账号' }}</button>
      </form>
    </template>
    <p v-else class="account-message">顾客账号需要通过 HTTPS 访问。</p>
    <p v-if="error" class="s-danger" role="alert">{{ error }}</p>
  </section>
</template>
