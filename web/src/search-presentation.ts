import type {Card} from './experience.ts';
import type {Work} from './types.ts';
// Shared with extension-kit/presentation/v1. No media-kind enum in this contract.
export type ResultPresentation={version:1;label:string;keywords?:string[];preview?:{aspectRatio:number;shape:'rectangle'|'circle';fit:'cover'|'contain'}};
const plain=(v:unknown):v is string=>typeof v==='string'&&!!v.trim()&&[...v].length<=40&&!/[\u0000-\u001f\u007f-\u009f]/u.test(v);
export function validPresentation(value:unknown):value is ResultPresentation{
 if(!value||typeof value!=='object')return false;
 const p=value as ResultPresentation;
 if(p.version!==1||!plain(p.label)||p.keywords!==undefined&&(!Array.isArray(p.keywords)||p.keywords.length>3||!p.keywords.every(plain)))return false;
 const v=p.preview;
 return v===undefined||!!v&&typeof v==='object'&&Number.isFinite(v.aspectRatio)&&v.aspectRatio>=.5&&v.aspectRatio<=3&&['rectangle','circle'].includes(v.shape)&&['cover','contain'].includes(v.fit)&&(v.shape!=='circle'||v.aspectRatio===1);
}
export function resolvePresentation(value:unknown,fallback:ResultPresentation):ResultPresentation{
 if(!validPresentation(value))return fallback;
 return {...value,label:value.label.trim(),keywords:[...new Set((value.keywords??[]).map(k=>k.trim()).filter(k=>k!==value.label.trim()))],preview:value.preview??fallback.preview};
}
export function searchPresentation(card:Card,works:Work[]=[]):ResultPresentation{
 const linked=works.find(w=>card.group?.entries.some(e=>e.workId===w.id));
 const md=card.work?.metadata??linked?.metadata??card.group?.entries.find(e=>e.metadata?.presentation)?.metadata??card.group?.entries[0]?.metadata??card.entry?.metadata;
 let label=card.kind==='films'?'Film':card.kind==='series'?'Série':card.work?'Sortie musicale':'Album';
 let ratio=card.kind==='music'?1:2/3;
 if(card.kind==='music'&&card.group&&!card.group.entries.some(e=>e.metadata?.album))label='Morceau';
 if(card.entry){label=card.kind==='music'?'Morceau':card.kind==='series'?'Épisode':'Film';if(card.kind==='series')ratio=16/9;}
 const fallback:ResultPresentation={version:1,label,preview:{aspectRatio:ratio,shape:'rectangle',fit:'cover'}};
 // A track's release metadata describes its album, not this individual track.
 return resolvePresentation(card.presentation??(card.entry?undefined:md?.presentation),fallback);
}
export function searchSecondary(p:ResultPresentation,subtitle:string){
 return [...new Set([p.label,...(p.keywords??[]),...subtitle.split(' · ')].map(s=>s.trim()).filter(Boolean))].join(' · ');
}
