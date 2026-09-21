import { describe, it, expect, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { defineComponent } from 'vue'
import OpenAIAccountProxyFields from '../OpenAIAccountProxyFields.vue'
vi.mock('vue-i18n', async () => ({ ...await vi.importActual<typeof import('vue-i18n')>('vue-i18n'), useI18n: () => ({ t: (key: string) => key }) }))
const Select = defineComponent({ props:['options','modelValue'], template:'<select><option v-for="o in options" :key="o.value" :value="o.value">{{o.label}}</option></select>' })
function render(id: number|null = null, groups: any[] = [{id:1,name:'ordinary',per_ip_concurrency:1,sticky_minutes:0},{id:2,name:'sticky',per_ip_concurrency:1,sticky_minutes:20}]) {
 return mount(OpenAIAccountProxyFields,{props:{platform:'openai',type:'oauth',proxyId:null,proxyIpGroupId:id,proxies:[],ipGroups:groups},global:{stubs:{Select,ProxySelector:true}}})
}
describe('sticky proxy selection',()=>{
 it('offers sticky mode and selects only its groups',async()=>{
  const w=render();const radios=w.findAll('input[type=radio]');expect(radios).toHaveLength(3)
  await radios[2].setValue(true)
  expect(w.emitted('update:proxyIpGroupId')?.at(-1)).toEqual([2])
  expect(w.findAll('option').map(x=>x.attributes('value'))).toEqual(['','2'])
 })
 it('restores saved sticky mode',()=>{
  const w=render(2);expect(w.findAll('input[type=radio]')).toHaveLength(3)
  expect((w.findAll('input[type=radio]')[2].element as HTMLInputElement).checked).toBe(true)
 })
 it('allows selecting an empty sticky mode without silently selecting a regular group',async()=>{
  const w=render(null,[]);const radios=w.findAll('input[type=radio]');expect(radios).toHaveLength(3)
  await radios[2].setValue(true)
  expect(w.find('select').exists()).toBe(true)
  expect(w.emitted('update:proxyIpGroupId')?.at(-1)).toEqual([null])
 })
})
