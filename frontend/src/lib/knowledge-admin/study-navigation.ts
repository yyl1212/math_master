const historyKeys=['knowledgeId','topicKey','kind','from','to','cursor','legacyCursor'] as const;

export function historyHref(filters:Record<string,string|string[]|undefined>={}){
 const query=new URLSearchParams({view:'history'});
 for(const key of historyKeys){const value=filters[key];if(typeof value==='string'&&value)query.set(key,value);}
 return '/learn?'+query;
}
