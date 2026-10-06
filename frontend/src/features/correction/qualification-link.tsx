"use client";
import {UiText} from "@/lib/i18n/ui-text";
import {uiMessage,uiValue} from "@/lib/i18n/format";
import Link from 'next/link';
import type {QualificationView} from '@/lib/learning/types';
export function QualificationLink({qualification:q}:{qualification:QualificationView}){return <p>{q.correctionId&&<><Link prefetch={false} href={'/corrections/'+q.correctionId}><UiText notice={uiMessage("qualification-link.view.corrected.qualification.742e9a",{})}/></Link> · </>}<Link prefetch={false} href={'/assessments/'+q.evidenceAttemptId+'/result'}><UiText notice={uiMessage("qualification-link.original.assessment.cebd35",{})}/></Link></p>}
