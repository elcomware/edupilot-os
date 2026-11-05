import {useState,type FormEvent} from 'react'
import {useMutation} from '@tanstack/react-query'
import {ArrowRight, KeyRound, Languages} from 'lucide-react'
import {api,useAuth} from '../../app/session'
import {t} from '../../app/copy'
import {errorKey} from '../../app/errors'
import {useLocale,usePreferences} from '../../app/preferences'
import {Brand} from '../../components/primitives'
export function Login(){
 const l=useLocale(),update=usePreferences(s=>s.update),start=useAuth(s=>s.start),expired=useAuth(s=>s.expired)
 const [school,setSchool]=useState(''),[email,setEmail]=useState(''),[password,setPassword]=useState('')
 const login=useMutation({mutationFn:api.login,onSuccess:start,onError:()=>setPassword('')})
 function submit(e:FormEvent){e.preventDefault();if(!login.isPending)login.mutate({school:school.trim(),email:email.trim(),password})}
 return <main className="login-page">
   <aside className="login-cover"><Brand/><div className="register-art" aria-hidden="true"><i/><i/><i/><i/></div>
    <div className="cover-caption"><span>{t(l,'brand')}</span><h2>{t(l,'register')}</h2></div></aside>
   <section className="login-pane"><div className="login-locale"><Languages aria-hidden="true" size={16}/><select aria-label={t(l,'language')} value={l} onChange={e=>update({locale:e.target.value==='fr'?'fr':'en'})}><option value="en">English</option><option value="fr">Français</option></select></div>
    <div className="login-box"><p className="eyebrow"><KeyRound size={16} aria-hidden="true"/>{t(l,'access')}</p><h1>{t(l,'welcome')}</h1><p className="lede">{t(l,'loginInfo')}</p>
    {expired&&<p role="status" className="notice">{t(l,'sessionExpired')}</p>}
    <form onSubmit={submit} className="stack">
     <label>{t(l,'school')}<input autoComplete="organization" required value={school} onChange={e=>setSchool(e.target.value)} aria-describedby="school-help" /></label>
     <small id="school-help" className="field-help">{t(l,'schoolHint')}</small>
     <label>{t(l,'email')}<input required autoComplete="username" type="email" value={email} onChange={e=>setEmail(e.target.value)} /></label>
     <label>{t(l,'password')}<input required autoComplete="current-password" type="password" value={password} onChange={e=>setPassword(e.target.value)} /></label>
     {login.error&&<p className="error-note" role="alert">{t(l,errorKey(login.error,true))}</p>}
     <button className="btn primary login-submit" disabled={login.isPending}>{t(l,login.isPending?'signingIn':'signIn')}<ArrowRight size={17}/></button>
    </form></div></section></main>
}
