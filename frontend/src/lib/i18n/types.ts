import type {enMessages} from "./messages/en";
export type MessageKey=keyof typeof enMessages;
export type MessageValues<K extends MessageKey>={readonly[P in (typeof enMessages)[K]["params"][number]]:string|number};
export type UiMessage={[K in MessageKey]:{kind:"system";key:K;values:MessageValues<K>}}[MessageKey];
export type UiErrorNamespace="public"|"auth"|"content"|"question"|"learning"|"feedback"|"correction"|"notification";
export type UiNotice=UiMessage|{kind:"error";namespace:UiErrorNamespace;code:string;requestId?:string;retryAfter?:number}|{kind:"literal";text:string};
