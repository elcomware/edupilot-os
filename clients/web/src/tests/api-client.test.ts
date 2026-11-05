import {describe,it,expect,vi} from 'vitest'
import {createApiClient} from '@edupilot/api-client'
const json=(data:unknown,status=200)=>new Response(JSON.stringify(data),{status,headers:{'Content-Type':'application/json'}})
describe('versioned transport',()=>{
 it('does not transmit an old bearer during sign-in or put credentials in the URL',async()=>{
   const fetcher=vi.fn(async()=>json({token:'new',principal:{id:'u',school:'a',permissions:[]}}))
   const api=createApiClient({baseUrl:'/api/v1/',getToken:()=>'old',fetcher})
   await api.login({school:'a',email:'staff@example.test',password:'secret'})
   expect(fetcher).toHaveBeenCalledWith('/api/v1/session',expect.objectContaining({method:'POST',headers:{Accept:'application/json','Content-Type':'application/json'}}))
 })
 it('submits the version and encodes the archive ID; no JSON parsing for 204',async()=>{
  const fetcher=vi.fn(async()=>new Response(null,{status:204}))
  const client=createApiClient({baseUrl:'/api/v1',getToken:()=>'token',fetcher})
  await expect(client.archiveStudent('a/b',7)).resolves.toBeUndefined()
  expect(fetcher).toHaveBeenCalledWith('/api/v1/students/a%2Fb?version=7',expect.objectContaining({method:'DELETE',headers:{Accept:'application/json',Authorization:'Bearer token'}}))
 })
 it('does not let a 401 from an earlier account sign out a new session',async()=>{
   let token='old'; const onUnauthorized=vi.fn()
   const api=createApiClient({baseUrl:'/api/v1',getToken:()=>token,onUnauthorized,fetcher:async()=>{token='new';return json({error:{code:'unauthenticated',message:'Sign in'}},401)}})
   await expect(api.session()).rejects.toMatchObject({status:401,code:'unauthenticated'});expect(onUnauthorized).not.toHaveBeenCalled()
 })
})
