const emit=(level,module,op,message)=>{if(process.env.NODE_ENV!=='production')window.dispatchEvent(new CustomEvent('suxu:merchant-diagnostic',{detail:{level,module,op,message}}))}
export const logger={info:(m,o,s)=>emit('info',m,o,s),warn:(m,o,s)=>emit('warn',m,o,s),error:(m,o,s)=>emit('error',m,o,s)}
