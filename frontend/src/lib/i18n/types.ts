import type {enMessages} from "./messages/en";
export type MessageKey=keyof typeof enMessages;
export type MessageValues<K extends MessageKey>={readonly[P in (typeof enMessages)[K]["params"][number]]:string|number};
export type StaticMessageKey={[K in MessageKey]:(typeof enMessages)[K]["params"][number] extends never?K:never}[MessageKey];
/** Opaque, compact DTO: uiMessage checks key-specific parameters at construction. */
declare const uiMessageBrand:unique symbol;
export type UiMessage={readonly kind:"system";readonly key:MessageKey;readonly values:Readonly<Record<string,string|number>>;readonly [uiMessageBrand]:true};
export type UiErrorNamespace="study"|"public"|"auth"|"content"|"question"|"learning"|"feedback"|"correction"|"notification";
export type UiNotice=UiMessage|{kind:"error";namespace:UiErrorNamespace;code:string;requestId?:string;retryAfter?:number}|{kind:"literal";text:string};
