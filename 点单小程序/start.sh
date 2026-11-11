#!/bin/sh
set -eu
cd "$(dirname "$0")"
mkdir -p .runtime
if [ ! -f backend/src/config/config.yaml ]; then
  printf '请将 backend/src/config/config.example.yaml 复制为 config.yaml，再填写 MySQL、JWT 和初始管理员配置。\n' >&2
  exit 1
fi
if [ ! -d frontend/node_modules ]; then (cd frontend && npm ci); fi
if [ ! -d admin/node_modules ]; then (cd admin && npm ci); fi
(cd backend && go build -o ../.runtime/suxu-api .)
api_pid=''
shop_pid=''
admin_pid=''
cleanup() {
  trap - INT TERM EXIT
  for child in "$admin_pid" "$shop_pid" "$api_pid"; do
    if [ -n "$child" ]; then kill "$child" 2>/dev/null || true; fi
  done
  wait 2>/dev/null || true
}
trap cleanup INT TERM EXIT
(cd backend && exec ../.runtime/suxu-api) &
api_pid=$!
ready=0
for attempt in 1 2 3 4 5 6 7 8 9 10 11 12 13 14 15; do
  if ! kill -0 "$api_pid" 2>/dev/null; then printf '后端启动失败，请检查上方日志及 YAML 配置。\n'; exit 1; fi
  if curl --noproxy '*' -fsS http://127.0.0.1:8090/healthz >/dev/null 2>&1; then ready=1; break; fi
  sleep 1
done
if [ "$ready" != 1 ]; then printf '后端未就绪，前后台未启动。\n'; exit 1; fi
(cd frontend && exec node node_modules/vite/bin/vite.js) &
shop_pid=$!
(cd admin && exec node node_modules/@vue/cli-service/bin/vue-cli-service.js serve) &
admin_pid=$!
printf '\n顾客商城： http://localhost:8081/\n独立管理端： http://localhost:8082/\n后端 API： http://127.0.0.1:8090/api/v1\nCtrl+C 停止此脚本启动的三个服务。\n\n'
wait
