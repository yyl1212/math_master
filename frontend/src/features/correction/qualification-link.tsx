import Link from 'next/link';
import type {QualificationView} from '@/lib/learning/types';
export function QualificationLink({qualification:q}:{qualification:QualificationView}){return <p>{q.correctionId&&<><Link prefetch={false} href={'/corrections/'+q.correctionId}>View corrected qualification</Link> · </>}<Link prefetch={false} href={'/assessments/'+q.evidenceAttemptId+'/result'}>Original assessment</Link></p>}
