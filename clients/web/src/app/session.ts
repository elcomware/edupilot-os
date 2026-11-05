import {create} from 'zustand'
import {QueryClient} from '@tanstack/react-query'
import {ApiError, createApiClient, type Session} from '@edupilot/api-client'
import {usePreferences} from './preferences'
export const queries=new QueryClient({defaultOptions:{queries:{retry:false,staleTime:30_000,networkMode:'always'},mutations:{retry:false,networkMode:'always'}}})
type Auth={session:Session|null;expired:boolean; revocationUnknown:boolean; epoch:number; start:(session:Session)=>void; end:(expired?:boolean)=>void}
export const useAuth=create<Auth>((set,get)=>({session:null,expired:false,revocationUnknown:false,epoch:0,
  start(session){queries.clear();usePreferences.getState().activate(session.principal);set({session,expired:false,revocationUnknown:false,epoch:get().epoch+1})},
  end(expired=false){queries.clear();usePreferences.getState().activate(null);set({session:null,expired,revocationUnknown:false})}
}))
export const api=createApiClient({baseUrl:import.meta.env.VITE_API_BASE_URL || '/api/v1',
  getToken:()=>useAuth.getState().session?.token,onUnauthorized:()=>useAuth.getState().end(true)})

// Capture the bearer in the request before clearing all local access. A late
// response from this logout must not affect a newer user's session.
export async function signOut(){
  const {epoch,end}=useAuth.getState()
  const revocation=api.logout()
  end()
  try {await revocation} catch(error) {
    const state=useAuth.getState()
    if(state.epoch===epoch && !state.session && !(error instanceof ApiError && error.status===401))
      useAuth.setState({revocationUnknown:true})
  }
}
