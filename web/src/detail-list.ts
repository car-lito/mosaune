import type {Entry} from './types';
export type DetailRow={id:string;title:string;ordinal:string;group:string};
export type DetailGroup={key:string;title:string;total:number;artwork?:number;requestSeason?:number};
export type DetailView={owner:string;title:string;all:string;files:string;help:string;empty:string;groups:DetailGroup[];rows:DetailRow[];images:string[]};
export function detailRows(view:DetailView|null,entries:Entry[],group:string|null){const byId=new Map(entries.map(e=>[e.id,e]));const rows=view?.rows??entries.map(e=>({id:e.id,title:e.metadata?.title||e.name,ordinal:'▶',group:''}));return rows.filter(r=>(group===null||r.group===group)&&byId.has(r.id)).map(r=>({...r,entry:byId.get(r.id)!}));}
export function groupCount(view:DetailView,group:string,entries:Entry[]){return detailRows(view,entries,group).length;}
