import {useRef,useState,type FormEvent} from 'react'
import {useMutation} from '@tanstack/react-query'
import {ArrowDownToLine,ArrowUpFromLine,Files} from 'lucide-react'
import {ApiError,type Principal} from '@edupilot/api-client'
import {api} from '../../app/session'
import {useLocale} from '../../app/preferences'
import {t} from '../../app/copy'
import {errorKey} from '../../app/errors'
import {sx} from './copy'
const extensions={'application/pdf':'pdf','text/plain':'txt','image/png':'png','image/jpeg':'jpg'}
export const allowedDocument=(f:Pick<File,'size'|'type'>)=>f.size>0&&f.size<=8*1024*1024&&Object.hasOwn(extensions,f.type)
export const validDocumentId=(id:string)=>/^[\da-f]{8}-[\da-f]{4}-[\da-f]{4}-[\da-f]{4}-[\da-f]{12}$/i.test(id.trim())
export function DocumentsPanel({principal}:{principal:Principal}){
 const l=useLocale(), canRead=principal.permissions.includes('documents.read'),canWrite=principal.permissions.includes('documents.write')
 const fileInput=useRef<HTMLInputElement>(null),[id,setId]=useState(''),[fileError,setFileError]=useState(false),[idError,setIdError]=useState(false)
 const upload=useMutation({mutationFn:(file:File)=>api.uploadDocument(file),onSuccess:result=>{setId(result.id);if(fileInput.current)fileInput.current.value='';download.reset()}})
 const download=useMutation({mutationFn:async(id:string)=>{
   const blob=await api.downloadDocument(id),url=URL.createObjectURL(blob),link=document.createElement('a')
   try{link.href=url;link.download=id+'.'+(extensions[blob.type.split(';')[0] as keyof typeof extensions]||'bin');document.body.append(link);link.click()}
   finally{link.remove();window.setTimeout(()=>URL.revokeObjectURL(url),1000)}
 }})
 if(!canRead&&!canWrite)return null
 function uploadFile(e:FormEvent){e.preventDefault();const file=fileInput.current?.files?.[0]
   setFileError(!file||!allowedDocument(file));if(!file||!allowedDocument(file))fileInput.current?.focus();if(file&&allowedDocument(file)&&!upload.isPending&&!unknown)upload.mutate(file)}
 function downloadFile(e:FormEvent){e.preventDefault();const valid=validDocumentId(id);setIdError(!valid);if(valid&&!download.isPending)download.mutate(id.trim())}
 const unknown=upload.error instanceof ApiError&&upload.error.code==='write_unknown'
 const error=(e:unknown)=>e instanceof ApiError && e.code==='write_unknown'?sx(l,'uploadUnknown'):e instanceof ApiError && e.status===503?sx(l,'unavailable'):e instanceof ApiError && e.status===415?sx(l,'unsupported'):e instanceof ApiError && e.status===413?sx(l,'chooseFile'):t(l,errorKey(e))
 return <section className="collection-sheet special-sheet"><div className="sheet-heading"><Files size={18}/><h2>{sx(l,'documents')}</h2></div><p className="panel-lede">{sx(l,'docsHelp')}</p>
 <div className="document-actions">{canWrite&&<form noValidate onSubmit={uploadFile} className="stack"><label>{sx(l,'file')}<input ref={fileInput} type="file" required accept={Object.keys(extensions).join(',')} aria-describedby={fileError?'format-hint file-error':'format-hint'} aria-invalid={fileError||undefined} disabled={upload.isPending} onChange={()=>{upload.reset();setFileError(false)}}/></label><small className="field-help" id="format-hint">{sx(l,'fileHelp')}</small>
 <button className="btn" disabled={upload.isPending||unknown}><ArrowUpFromLine size={16}/>{sx(l,upload.isPending?'uploading':'upload')}</button>
 {fileError&&<p id="file-error" role="alert" className="error-note">{sx(l,'chooseFile')}</p>}{upload.error&&<p role="alert" className="error-note">{error(upload.error)}</p>}
 {upload.data&&<p className="notice" role="status">{sx(l,'uploaded')} <code>{upload.data.id}</code></p>}</form>}
 {canRead&&<form noValidate className="stack" onSubmit={downloadFile}><label>{sx(l,'id')}<input required value={id} disabled={download.isPending} aria-invalid={idError||undefined} aria-describedby={idError?'id-hint id-error':'id-hint'} autoComplete="off" onChange={e=>{setId(e.target.value);setIdError(false);download.reset()}}/></label><small className="field-help" id="id-hint">{sx(l,'idHint')}</small>
 <button className="btn" disabled={download.isPending}><ArrowDownToLine size={16}/>{sx(l,download.isPending?'downloading':'download')}</button>
 {idError&&<p id="id-error" className="error-note" role="alert">{sx(l,'idRule')}</p>}{download.error&&<p className="error-note" role="alert">{error(download.error)}</p>}{download.isSuccess&&<p role="status">{sx(l,'downloaded')}</p>}</form>}</div></section>
}
