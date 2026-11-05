import {ApiError} from '@edupilot/api-client'
import type {CopyKey} from './copy'
export function errorKey(error:unknown,login=false):CopyKey {
  if(!(error instanceof ApiError))return 'network'
  if(login && (error.status===401||error.status===403))return 'invalidLogin'
  switch(error.status){case 401:return 'sessionExpired';case 403:return 'denied';case 404:return 'missing';case 409:case 412:return 'stale';case 422:case 400:return 'badInput';case 429:return 'rate';default:return 'server'}
}
