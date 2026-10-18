import test from 'node:test';
import assert from 'node:assert/strict';
import {searchPresentation,resolvePresentation,searchSecondary,validPresentation} from './search-presentation.ts';

test('search distinguishes local groups, individual tracks and episodes without guessing single from track count',()=>{
 const card={kind:'music',group:{entries:[{metadata:{title:'One song',album:'Record'}}]}};
 assert.equal(searchPresentation(card).label,'Album');
 assert.equal(searchPresentation(card).preview.aspectRatio,1);
 assert.equal(searchPresentation({kind:'films'}).label,'Film');
 assert.equal(searchPresentation({kind:'series',group:{entries:[{metadata:{episode:1}}]}}).label,'Série');
 assert.equal(searchPresentation({kind:'series',entry:{metadata:{episode:1}}}).label,'Épisode');
 assert.equal(searchPresentation({kind:'series',entry:{}}).preview.aspectRatio,16/9);
 assert.equal(searchPresentation({kind:'music',entry:{metadata:{presentation:{version:1,label:'Album'}}}}).label,'Morceau');
 assert.equal(searchPresentation({kind:'music',work:{metadata:{}}}).label,'Sortie musicale');
});
test('metadata labels survive local and online search and extensions define new types without editing the renderer',()=>{
 const single={version:1,label:'Single',preview:{aspectRatio:1,shape:'rectangle',fit:'cover'}};
 assert.equal(searchPresentation({kind:'music',work:{metadata:{presentation:single}}}).label,'Single');
 assert.equal(searchPresentation({kind:'music',group:{entries:[{metadata:{presentation:single}}]}}).label,'Single');
 assert.equal(searchPresentation({kind:'music',group:{entries:[{workId:'linked',metadata:{album:'Record'}}]}},[{id:'linked',metadata:{presentation:single}}]).label,'Single');
 const game={version:1,label:'Jeu',keywords:['Console','Nintendo Switch'],preview:{aspectRatio:.62,shape:'rectangle',fit:'contain'}};
 assert.deepEqual(resolvePresentation(game,single),game);
 assert.equal(searchSecondary(game,'2026'),'Jeu · Console · Nintendo Switch · 2026');
 assert.equal(searchSecondary({version:1,label:'Série',keywords:['Série']},'Série'),'Série');
 for(const label of ['Artiste','Acteur','Saison','Console'])assert.ok(validPresentation({version:1,label,preview:{aspectRatio:1,shape:'circle',fit:'cover'}}));
});
test('invalid presentation falls back without injecting CSS; labels remain text, not HTML',()=>{
 const fallback={version:1,label:'Film',preview:{aspectRatio:2/3,shape:'rectangle',fit:'cover'}};
 for(const bad of [null,{}, {version:2,label:'Jeu'},{version:1,label:'\nFilm'},{version:1,label:'x'.repeat(41)},{version:1,label:'Jeu',keywords:['1','2','3','4']},...['1; background:url(x)',0,Infinity,NaN,4].map(aspectRatio=>({version:1,label:'Jeu',preview:{aspectRatio,shape:'rectangle',fit:'cover'}})),{version:1,label:'Acteur',preview:{aspectRatio:2,shape:'circle',fit:'cover'}}])assert.deepEqual(resolvePresentation(bad,fallback),fallback);
 assert.equal(resolvePresentation({version:1,label:'<b>Jeu</b>'},fallback).label,'<b>Jeu</b>');
});
