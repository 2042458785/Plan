<template>
  <span>
    <el-button plain icon="el-icon-full-screen" @click="open">选购二维码</el-button>
    <el-dialog title="扫一扫，发现心选之物" :visible.sync="visible" width="410px" append-to-body custom-class="qr-dialog">
      <div class="qr-brand">SUXU</div>
      <img v-if="image" :src="image" alt="顾客选购二维码" class="qr-image">
      <el-form label-position="top" @submit.native.prevent="generate">
        <el-form-item label="顾客入口">
          <el-radio-group v-model="destination" size="small" @change="selectDestination">
            <el-radio-button label="store">完整商城</el-radio-button>
            <el-radio-button label="quick">快速小程序</el-radio-button>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="顾客商城网址">
          <el-input v-model="url" placeholder="http://你的商城地址/" @input="image = ''"></el-input>
        </el-form-item>
        <div class="dialog-actions">
          <el-button native-type="submit">更新二维码</el-button>
          <a v-if="image" :href="image" download="素序选购二维码.png" class="el-button el-button--primary">下载二维码</a>
        </div>
      </el-form>
    </el-dialog>
  </span>
</template>

<script>
import QRCode from 'qrcode'
import { APP_CONFIG } from '@/config'

export default {
  data: () => ({ visible: false, destination: 'store', url: APP_CONFIG.storefrontURL, image: '' }),
  methods: {
    open() {
      this.visible = true
      if (this.url) this.generate()
    },
    selectDestination(destination) {
      this.url = destination === 'quick' ? APP_CONFIG.quickStorefrontURL : APP_CONFIG.storefrontURL
      this.generate()
    },
    async generate() {
      const input = this.url
      this.image = ''
      try {
        const url = new URL(input)
        if (!['http:', 'https:'].includes(url.protocol)) throw new Error('invalid protocol')
        const image = await QRCode.toDataURL(url.href, {
          width: 400,
          margin: 2,
          color: { dark: '#35382b', light: '#fffefa' },
        })
        if (input === this.url) this.image = image
      } catch {
        if (input === this.url) this.$message.error('请填写完整的 http 或 https 商城网址')
      }
    },
  },
}
</script>
