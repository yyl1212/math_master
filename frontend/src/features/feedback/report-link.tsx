import Link from 'next/link';
import type {ContextQuery} from '@/lib/feedback/types';
export function ReportLink({source}:{source:ContextQuery}) {
  const params=new URLSearchParams();
  for(const key of ['kind','id','area','position','partKind','partId'] as const) if(source[key]!==undefined)params.set(key,String(source[key]));
  return <Link prefetch={false} href={'/feedback/new?'+params} className="button secondary">Report a problem</Link>;
}
