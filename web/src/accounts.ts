import type {Entry,Kind} from './types';
export type Grant={view:boolean;play:boolean;edit:boolean;request:boolean};
export type Account={id:string;name:string;admin:boolean;disabled:boolean;pending:boolean;kinds:Kind[];libraries:Record<string,Grant>;theme:'light'|'dark'|'system'};
export let currentAccount:Account|null=null;
export function setAccount(a:Account|null){currentAccount=a;}
export function permitted(e:Entry|undefined,action:keyof Grant){if(!e)return false;if(!e.libraryId)return true;const a=currentAccount;if(!a)return false;if(a.admin)return true;const g=a.libraries[e.libraryId];return a.kinds.includes(e.kind)&&!!g?.view&&!!g[action];}
export function canRequest(kind:Kind){const a=currentAccount;return !!a&&(a.admin||a.kinds.includes(kind)&&Object.values(a.libraries).some(g=>g.view&&g.request));}
export async function accountRequest(token:string,path:string,method='GET',body?:unknown){const r=await fetch('/api/v1/'+path,{method,headers:{'Content-Type':'application/json','X-Mosaune-Token':token},body:body===undefined?undefined:JSON.stringify(body)});if(r.status===204)return null;const data=await r.json();if(!r.ok)throw Error(data.error||'Opération indisponible.');return data;}
export function applyTheme(theme:Account['theme']){document.documentElement.dataset.theme=theme==='system'?(matchMedia('(prefers-color-scheme: dark)').matches?'dark':'light'):theme;}
