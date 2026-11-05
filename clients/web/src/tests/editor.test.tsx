// @vitest-environment jsdom
import {beforeAll,afterEach,describe,it,expect,vi} from 'vitest'
import {render,screen,cleanup} from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import {QueryClient,QueryClientProvider} from '@tanstack/react-query'
import {ApiError} from '@edupilot/api-client'
import {api} from '../app/session'
import {RecordDialog} from '../features/collections/RecordDialog'
beforeAll(()=>{
 HTMLDialogElement.prototype.showModal=function(){this.setAttribute('open','')}
 HTMLDialogElement.prototype.close=function(){this.removeAttribute('open')}
})
afterEach(()=>{cleanup();vi.restoreAllMocks()})
describe('versioned student edits',()=>{
 it('retains a rejected draft and adopts the latest version only when requested',async()=>{
 const row={id:'r1',givenName:'Jo',familyName:'Bell',status:'active',version:2}
 const latest=vi.fn(async()=>({...row,givenName:'Joseph',version:3}))
 const update=vi.spyOn(api,'updateStudent').mockRejectedValueOnce(new ApiError(409,'version_conflict','Changed')).mockResolvedValue({...row,version:4})
 const onSaved=vi.fn(),user=userEvent.setup()
 render(<QueryClientProvider client={new QueryClient()}><RecordDialog module="students" editor={{kind:'edit',row}} onSaved={onSaved} onClose={()=>{}} latest={latest}/></QueryClientProvider>)
 await user.clear(screen.getByLabelText('Given name'));await user.type(screen.getByLabelText('Given name'),'Joe')
 await user.click(screen.getByRole('button',{name:'Save changes'}))
 await screen.findByRole('alert');expect((screen.getByLabelText('Given name') as HTMLInputElement).value).toBe('Joe');expect(onSaved).not.toHaveBeenCalled()
 await user.click(screen.getByRole('button',{name:'Load latest version'}));await screen.findByDisplayValue('Joseph')
 await user.click(screen.getByRole('button',{name:'Save changes'}))
 expect(update).toHaveBeenLastCalledWith('r1',{givenName:'Joseph',familyName:'Bell',version:3})
 })
})
