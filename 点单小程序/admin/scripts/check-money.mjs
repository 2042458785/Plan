import fs from 'node:fs'
import assert from 'node:assert/strict'
const source=fs.readFileSync(new URL('../src/utils/money.js',import.meta.url),'utf8')
const {money,parseYuan}=await import(`data:text/javascript;base64,${Buffer.from(source).toString('base64')}`)
assert.equal(parseYuan('699.99'),69999)
assert.equal(parseYuan('0.01'),1)
assert.equal(parseYuan('9.9'),990)
assert.equal(money(69999),'699.99')
assert.equal(money(-1),'-0.01')
for(const value of ['1.999','-1','0','NaN','10000.01'])assert.throws(()=>parseYuan(value))
process.stdout.write('PASS：金额按整数分解析与展示，拒绝隐式四舍五入。\n')
