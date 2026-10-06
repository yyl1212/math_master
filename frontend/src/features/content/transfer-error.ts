import {uiError} from "@/lib/i18n/errors";
import type {UiNotice} from "@/lib/i18n/types";
/** Keeps the original error message for callers; UI uses its explicit code. */
export class ContentTransferError extends Error {constructor(message:string,readonly code:string){super(message);}}
export function transferNotice(error:unknown):UiNotice{return uiError("content",{code:error instanceof ContentTransferError?error.code:"TRANSFER_UNAVAILABLE"});}
