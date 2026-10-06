import {z} from "zod";
import {parseContentJSON,draftInputSchema} from "@/lib/content/schemas";
import {assignmentInputSchema} from "@/lib/taxonomy/schemas";
import {importDraftInput} from "./asset-transfer";
import {ContentTransferError} from "./transfer-error";
const schema=z.object({kind:z.literal("topic-draft"),schemaVersion:z.literal(1),draft:draftInputSchema,assignments:z.array(assignmentInputSchema).max(100),sourceBatchSHA:z.string().regex(/^[a-f0-9]{64}$/)}).strict();
export function importTopicDraftEnvelope(bytes:Uint8Array){
 if(bytes.byteLength>2<<20)throw new ContentTransferError("This content exceeds the request size limit.","PAYLOAD_TOO_LARGE");
 const value=schema.safeParse(parseContentJSON(bytes));if(!value.success)throw new ContentTransferError("Invalid topic draft envelope.","TRANSFER_JSON_INVALID");const envelope=value.data;
 const seen=new Set<string>();if(envelope.assignments.length!==envelope.draft.package.knowledge.length||envelope.assignments.some(m=>{if(seen.has(m.knowledge.id)||m.sourceBatchSHA!==envelope.sourceBatchSHA||!envelope.draft.package.knowledge.some(k=>k.id===m.knowledge.id&&k.version===m.knowledge.version))return true;seen.add(m.knowledge.id);return false}))throw new ContentTransferError("Invalid topic draft members.","TRANSFER_JSON_INVALID");
 const draft=importDraftInput(new TextEncoder().encode(JSON.stringify(envelope.draft)));return {...envelope,draft};
}
