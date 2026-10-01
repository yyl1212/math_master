import "server-only";
import {createHash} from "node:crypto";
import {validSVGMarkup} from "./svg-markup";
export function validateContentSVG(bytes:Uint8Array,expectedSHA:string):boolean{
 return bytes.byteLength<=1<<20&&/^[0-9a-f]{64}$/.test(expectedSHA)&&createHash("sha256").update(bytes).digest("hex")===expectedSHA&&validSVGMarkup(bytes);
}
