import "server-only";
import {headers} from "next/headers";
import {parseUiLocaleCookie,type UiLocale} from "./config";
export async function readRequestUiLocale():Promise<UiLocale>{try{return parseUiLocaleCookie((await headers()).get("cookie")??"");}catch{return "en";}}
