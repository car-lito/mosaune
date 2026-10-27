export type HeaderView={owner:string;title:string;eyebrow:string;facts:{key:string;text:string;note?:string}[];tagline:string;footnote:string};
// These are the only facts sent by the browser: identity, never metadata text.
export function headerRequest(kind:string,workId:string,ids:string[],trackId=''){return JSON.stringify({kind,workId,ids,trackId})}
