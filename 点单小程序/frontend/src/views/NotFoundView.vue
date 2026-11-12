<template>
  <main class="missing-page">
    <h1>页面未找到</h1>
    <router-link to="/">返回素序首页 <span aria-hidden="true">→</span></router-link>
  </main>
</template>

<style scoped>
.missing-page { display: grid; place-content: center; justify-items: center; gap: 18px; min-height: 100dvh; padding: 20px; text-align: center; background: #fffefa; color: #514530; }
.missing-page h1 { font-size: clamp(23px, 5vw, 32px); font-weight: 500; }
.missing-page a { display: inline-flex; align-items: center; gap: 12px; min-height: 44px; padding: 8px 16px; border: 1px solid #d5c7ae; border-radius: 10px; }
.missing-page a:focus-visible { outline: 2px solid #776240; outline-offset: 3px; }
</style>
