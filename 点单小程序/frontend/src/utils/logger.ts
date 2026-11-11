// 统一日志入口。生产不输出调试消息，也不记录令牌、联系方式或订单正文。
const emit = (level:string,module:string,op:string,message:string) => {
  if (import.meta.env.DEV) window.dispatchEvent(new CustomEvent('suxu:diagnostic',{detail:{level,module,op,message}}))
}
export const logger = {
  info:(module:string,op:string,message:string)=>emit('info',module,op,message),
  warn:(module:string,op:string,message:string)=>emit('warn',module,op,message),
  error:(module:string,op:string,message:string)=>emit('error',module,op,message),
}
