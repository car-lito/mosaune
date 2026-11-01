import test from 'node:test';
import assert from 'node:assert/strict';
import {localSearch,mergeSearch,normalizeSearch} from './catalogue-search.ts';
const presentation=(label='Album',ratio=1)=>({version:1,label,preview:{aspectRatio:ratio,shape:'rectangle',fit:'cover'}});
const spec=(target,title,terms=[],priority=40,label='Album')=>({target,title,subtitle:'Artist',terms,priority,presentation:presentation(label)});
const entry=(id,index,key='album')=>({id,name:'raw-'+id,filename:id,kind:'music',url:'local',size:1,catalogue:{owner:'contributor',key,title:'Album',subtitle:'Artist',order:[0,0],sortText:id,search:index},metadata:{title:id,album:'Album',artist:'Artist'}});
const index=(...entries)=>({version:1,entries});
test('extension candidates group albums and keep individual tracks with stable identities',()=>{
 const entries=['one','two'].map(id=>entry(id,index(spec('group','Album',['Artist',id]),spec('item',id,['Album'],30,'Morceau'))));
 const results=localSearch(entries,['music'],'album');
 assert.equal(results.length,3);assert.equal(results[0].title,'Album');assert.equal(results.filter(c=>c.group).length,1);assert.equal(results.filter(c=>c.entry).length,2);
 assert.equal(localSearch(entries,['films'],'album').length,0);
});
test('matching is accent insensitive, token based and exact titles outrank keyword priorities',()=>{
 assert.equal(normalizeSearch('Été — ŒUVRE'),'ete œuvre');
 const entries=[entry('a',index(spec('item','Été',['rive gauche'],0))),entry('b',index(spec('item','Other',['ete'],100)))];
 assert.equal(localSearch(entries,['music'],'ete')[0].title,'Été');
 assert.equal(localSearch(entries,['music'],'gauche ETE')[0].entry.id,'a');
 assert.equal(localSearch(entries,['music'],'missing').length,0);
});
test('a contributor supplies new keywords, type labels and previews without media-specific rendering',()=>{
 const game=spec('item','My game',['console','portable'],80,'Jeu');game.presentation.preview={aspectRatio:1,shape:'circle',fit:'contain'};
 const result=localSearch([entry('one',index(game))],['music'],'console')[0];
 assert.equal(result.presentation.label,'Jeu');assert.equal(result.presentation.preview.shape,'circle');assert.equal(result.entry.id,'one');assert.equal(result.image,undefined);
});
test('missing legacy index gives only filenames while an explicit empty index gives no results',()=>{
 const old=entry('one',undefined);assert.equal(localSearch([old],['music'],'album').length,0);
 assert.equal(localSearch([old],['music'],'raw')[0].presentation.label,'Fichier');
 old.catalogue.search=index();assert.equal(localSearch([old],['music'],'raw').length,0);
 delete old.catalogue;assert.equal(localSearch([old],['music'],'raw').length,1);
});
test('series and episodes are candidates with independent shapes, not inferred by the search renderer',()=>{
 const group=spec('group','Show',[],60,'Série'),ep=spec('item','Pilot',['Show'],40,'Épisode');group.presentation=presentation('Série',2/3);ep.presentation=presentation('Épisode',16/9);
 const item=entry('e1',index(group,ep),'show');item.kind='series';
 const results=localSearch([item],['series'],'show');assert.equal(results.length,2);assert.equal(results[1].presentation.preview.aspectRatio,16/9);
});
test('provider metadata only refines presentation when the extension explicitly opts in',()=>{
 const album=spec('group','Album',[],40),item=entry('one',index(album));item.workId='w';
 const works=[{id:'w',metadata:{presentation:presentation('Single')}}];
 assert.equal(localSearch([item],['music'],'album',works)[0].presentation.label,'Album');
 album.metadataPresentation=true;assert.equal(localSearch([item],['music'],'album',works)[0].presentation.label,'Single');
});
test('remote deduplication checks local groups AND tracks without merging different providers',()=>{
 const item=entry('one',index(spec('item','One',[],40)));item.metadata.provider='p';item.metadata.providerId='1';
 const local=localSearch([item],['music'],'one');
 const work=(id,provider,pid)=>({id,kind:'music',metadata:{title:id,provider,providerId:pid},localIds:[]});
 const remote=[work('a','p','1'),work('b','other','1'),work('c','other','1')];
 assert.deepEqual(mergeSearch(local,remote).map(c=>c.key),['b']);
});
