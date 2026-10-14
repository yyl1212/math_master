// Match Go strings.TrimSpace / unicode.IsSpace, including NEL and excluding BOM.
export function isReviewAnswerBlank(value:string):boolean{
 return /^[\u0009-\u000d\u0020\u0085\u00a0\u1680\u2000-\u200a\u2028\u2029\u202f\u205f\u3000]*$/.test(value);
}
