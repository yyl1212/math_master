import {FeedbackRequestError} from "@/lib/feedback/types";
export function topicFeedbackLocation(q:Record<string,string|string[]|undefined>):string{
 if(q.topicId===undefined&&q.taxonomyVersionId===undefined)return "";
 if(q.kind!=="site"||q.area!=="other"||typeof q.topicId!=="string"||!/^msc-\d{2}(?:[a-z](?:\d{2})?|-\d{2})?$/.test(q.topicId)||typeof q.taxonomyVersionId!=="string"||!/^[a-f0-9]{64}$/.test(q.taxonomyVersionId))throw new FeedbackRequestError("INVALID_REQUEST");
 return "MSC2020 "+q.topicId+" @ "+q.taxonomyVersionId;
}
