import {create} from 'zustand'
import {QueryClient} from '@tanstack/react-query'
import {createApiClient, type Session} from '@edupilot/api-client'
import {usePreferences} from './preferences'
export const queries=new QueryClient({defaultOptions:{queries:{retry:false,staleTime:30_000},mutations:{retry:false}}})
type Auth={session:Session|null;expired:boolean; start:(session:Session)=>void; end:(expired?:boolean)=>void}
export const useAuth=create<Auth>((set)=>({session:null,expired:false,
  start(session){queries.clear();usePreferences.getState().activate(session.principal);set({session,expired:false})},
  end(expired=false){queries.clear();usePreferences.getState().activate(null);set({session:null,expired})}
}))
export const api=createApiClient({baseUrl:import.meta.env.VITE_API_BASE_URL || '/api/v1',
  getToken:()=>useAuth.getState().session?.token,onUnauthorized:()=>useAuth.getState().end(true)})
