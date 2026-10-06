"use client";
import {useUiI18n} from "@/lib/i18n/provider";
import {UiText} from "@/lib/i18n/ui-text";
import {uiError} from "@/lib/i18n/errors";
import type {useStudyCommand} from "./pending-command";
export function StudyCommandFeedback({command}:{command:Pick<ReturnType<typeof useStudyCommand>,"busy"|"error"|"pending"|"retrySameRequest">}){const{t}=useUiI18n();return <div aria-live="polite">{command.busy&&<p role="status">{t("study.saving",{})}</p>}{command.error&&<div role="alert"><p>{command.error.code==="SERVICE_UNAVAILABLE"?t("study.uncertain",{}):<UiText notice={uiError("study",command.error)}/>}</p>{command.pending&&<button type="button" className="button secondary" disabled={command.busy} onClick={()=>void command.retrySameRequest()}>{t("study.retry",{})}</button>}</div>}</div>}
