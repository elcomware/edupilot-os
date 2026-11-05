import {useSyncExternalStore} from 'react'
type Route = {page:'workspace'|'settings'|'unknown'} | {page:'module';key:string}
export function readRoute(hash:string):Route {
  if(hash==='' || hash==='#/' || hash==='#/workspace') return {page:'workspace'}
  if(hash==='#/settings')return {page:'settings'}
  const match=/^#\/modules\/([a-z0-9][a-z0-9_-]*)$/.exec(hash)
  return match?{page:'module',key:match[1]}:{page:'unknown'}
}
const subscribe=(cb:()=>void)=>{window.addEventListener('hashchange',cb);return()=>window.removeEventListener('hashchange',cb)}
export function useRoute() {return readRoute(useSyncExternalStore(subscribe,()=>window.location.hash,()=>''))}
export const moduleHref=(key:string)=>'#/modules/'+encodeURIComponent(key)
