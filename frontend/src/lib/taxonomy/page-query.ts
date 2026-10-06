import {validContentString} from "../content/raw-json";
import {normalizeTopicQuery} from "./protocol";
import type {TopicQuery,PairRef} from "./types";
export function parseTopicPageQuery(params:Record<string,string|string[]|undefined>):({ok:true;query:TopicQuery;q:string;level?:number;kind:"primary"|"auxiliary"|"other"}|{ok:false}){
 if(Object.entries(params).some(([key,v])=>!["q","level","kind","offset","status"].includes(key)||Array.isArray(v)))return {ok:false};if(params.status!==undefined&&(typeof params.status!=="string"||!["all","planned","published"].includes(params.status)))return {ok:false};const q=params.q??"",kind=params.kind??"primary";if(typeof q!=="string"||typeof kind!=="string"||!validContentString(q))return {ok:false};const query:TopicQuery={limit:100};if(q)query.q=q;
 if(kind!=="primary"&&kind!=="auxiliary"&&kind!=="other")return {ok:false};query.kind=kind;
 for(const name of ["offset","level"] as const){const value=params[name];if(value!==undefined){if(typeof value!=="string"||!/^[0-9]+$/.test(value))return {ok:false};if(name==="level"&&value==="0")continue;query[name]=Number(value)}}
 if(!normalizeTopicQuery(query))return {ok:false};return {ok:true,query,q,level:query.level,kind}
}
export const sameTopicPair=(a:PairRef,b:PairRef)=>a.knowledgeHead===b.knowledgeHead&&a.taxonomyHead===b.taxonomyHead&&a.taxonomyVersionId===b.taxonomyVersionId;
export function parseTopicDetailOffsets(params:Record<string,string|string[]|undefined>):{offset:number;knowledgeOffset:number}|null{
 if(Object.entries(params).some(([key,v])=>!["offset","knowledgeOffset"].includes(key)||Array.isArray(v)))return null;const out={offset:0,knowledgeOffset:0};for(const key of ["offset","knowledgeOffset"] as const){const v=params[key];if(v!==undefined){if(typeof v!=="string"||!/^[0-9]+$/.test(v)||Number(v)>100000)return null;out[key]=Number(v)}}return out;
}
