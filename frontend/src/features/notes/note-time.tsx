'use client';import {useEffect,useState} from 'react';import {useUiI18n} from '@/lib/i18n/provider';
export function NoteTime({value}:{value:string}){const{locale}=useUiI18n(),[zone,setZone]=useState('UTC');useEffect(()=>setZone(Intl.DateTimeFormat().resolvedOptions().timeZone),[]);return <time dateTime={value}>{new Date(value).toLocaleString(locale,{timeZone:zone})}</time>}
