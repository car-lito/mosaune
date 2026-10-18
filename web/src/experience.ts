import { groupMedia, type Group } from './collection.ts';
import type { Entry, Work, Kind, CatalogTrack } from './types.ts';
import type {ResultPresentation} from './search-presentation.ts';
export type Card = {presentation?:ResultPresentation;key:string;title:string;subtitle:string;kind:Kind;image?:string;fallbackImage?:string;local:boolean;work?:Work;group?:Group;entry?:Entry;position?:number;duration?:number};
export const workCard=(w:Work):Card=>({key:w.id,title:w.metadata.title,subtitle:[w.metadata.artist,w.metadata.year].filter(Boolean).join(' · '),kind:w.kind,image:w.posterUrl||(w.poster?`/api/v1/discovery/works/${w.id}/poster`:undefined),local:w.localIds.length>0,work:w});
export const groupCard=(g:Group):Card=>({key:g.key,title:g.title,subtitle:g.subtitle,kind:g.kind,image:g.entries.find(e=>e.posterUrl)?.posterUrl||(g.entries.find(e=>e.workId)?.workId?`/api/v1/discovery/works/${g.entries.find(e=>e.workId)!.workId}/poster`:g.imageId?`/api/v1/artwork/${g.imageId}`:undefined),fallbackImage:g.imageId?`/api/v1/artwork/${g.imageId}`:undefined,local:true,group:g});
export function recentGroups(entries:Entry[],kind:Kind){return groupMedia(entries.filter(e=>e.kind===kind)).sort((a,b)=>Math.max(...b.entries.map(e=>e.addedAt??0))-Math.max(...a.entries.map(e=>e.addedAt??0))).map(groupCard);}
export function unfinished(entries:Entry[]):Card[]{return entries.filter(e=>e.libraryId&&!e.completed&&(e.position??0)>0&&(e.duration??0)>0).sort((a,b)=>(b.lastPlayed??0)-(a.lastPlayed??0)).map(e=>({key:e.id,title:e.metadata?.series||e.metadata?.title||e.name,subtitle:e.kind==='series'?`S${e.metadata?.season??0} · E${e.metadata?.episode??0} — ${e.metadata?.title??''}`:e.metadata?.artist??'À reprendre',kind:e.kind,image:e.posterUrl||(e.metadata?.imageId?`/api/v1/artwork/${e.metadata.imageId}`:undefined),local:true,entry:e,position:e.position,duration:e.duration}));}
export const normalizeTitle=(s:string)=>s.normalize('NFKD').replace(/\p{M}/gu,'').toLocaleLowerCase('fr').replace(/[^\p{L}\p{N}]+/gu,' ').trim();
export function trackEntry(track:CatalogTrack,entries:Entry[]):Entry|undefined{
 const matches=entries.filter(e=>normalizeTitle(e.metadata?.title||e.name)===normalizeTitle(track.title));
 const numbered=matches.filter(e=>(e.metadata?.disc||1)===(track.disc||1)&&e.metadata?.track===track.number);
 return numbered.length>=1?numbered[0]:matches.length===1?matches[0]:undefined;
}
export const timeLabel=(seconds:number)=>{const n=Math.max(0,Math.floor(seconds));return `${Math.floor(n/60)}:${String(n%60).padStart(2,'0')}`;};
