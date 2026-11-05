import type {ModuleDescriptor,Principal,Student,WorkspaceEntry} from '@edupilot/api-client'
export type RegisterRow = Student | WorkspaceEntry
export const isStudent=(row:RegisterRow):row is Student=>'givenName' in row
export const rowName=(row:RegisterRow)=>isStudent(row)?[row.givenName,row.familyName].join(' '):row.title
export function collectionAccess(module:ModuleDescriptor,principal:Principal) {
  const prefix=module.key==='students'?'students':'workspace'
  const can= (verb:string, permission:string) => module.capabilities.includes(verb) && principal.permissions.includes(prefix+'.'+permission)
  return {list:can('list','read'),create:can('create','write'),edit:can('update','write'),archive:module.key==='students'&&can('archive','write')}
}
export const PAGE_SIZE=25
export function normalizeText(value:string,limit:number,{allowControls=false}={}) {
  const clean=value.trim()
  return clean && Array.from(clean).length<=limit && (allowControls || !/[\u0000-\u001f\u007f-\u009f]/u.test(clean))?clean:null
}
