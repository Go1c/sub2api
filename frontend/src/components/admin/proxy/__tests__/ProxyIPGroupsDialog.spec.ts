import { describe, it, expect, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { defineComponent } from 'vue'
const api = vi.hoisted(() => ({ list:vi.fn().mockResolvedValue([]), create:vi.fn().mockResolvedValue({id:1}), update:vi.fn(), setMembers:vi.fn() }))
vi.mock('@/api/admin',()=>({adminAPI:{proxyIpGroups:api,proxies:{getAll:vi.fn().mockResolvedValue([])}}}))
vi.mock('@/stores/app',()=>({useAppStore:()=>({showError:vi.fn(),showSuccess:vi.fn()})}))
vi.mock('vue-i18n',async()=>({...await vi.importActual<typeof import('vue-i18n')>('vue-i18n'),useI18n:()=>({t:(s:string)=>s})}))
import Dialog from '../ProxyIPGroupsDialog.vue'
const BaseDialog=defineComponent({props:['show'],template:'<section v-if="show"><slot/><slot name="footer"/></section>'})
describe('sticky group form',()=>{
 it('creates a sticky group with the selected hold time',async()=>{
  const w=mount(Dialog,{props:{show:false},global:{stubs:{BaseDialog,ConfirmDialog:true,Icon:true}}})
  await w.setProps({show:true});await flushPromises()
  await w.get('button.btn-primary').trigger('click')
  await w.get('input[type=text]').setValue('sticky-test')
  await w.get('input[type=checkbox]').setValue(true)
  const inputs=w.findAll('input[type=number]')
  await inputs[1].setValue(30)
  await w.get('form').trigger('submit');await flushPromises()
  expect(api.create).toHaveBeenCalledWith({name:'sticky-test',per_ip_concurrency:10,sticky_minutes:30,proxy_ids:[]})
 })
})
