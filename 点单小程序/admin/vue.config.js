const { defineConfig } = require('@vue/cli-service')
module.exports = defineConfig({
  transpileDependencies: true,
  outputDir: 'dist_end',
  publicPath: process.env.NODE_ENV === 'production' ? '/admin/' : '/',
  productionSourceMap: false,
  devServer: {
    host: '0.0.0.0', port: 8082,
    proxy: {
      '/api': { target: 'http://127.0.0.1:8090', changeOrigin: false },
    },
  },
})
