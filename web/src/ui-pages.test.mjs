import {test} from 'node:test';import assert from 'node:assert/strict';
import {pageKey,collectionCards} from './ui-pages.ts';
test('extension routes are namespaced and cannot replace host settings',()=>{assert.equal(pageKey({id:'account'}),'account');assert.equal(pageKey({id:'account',owner:'atelier.salon'}),'extension:atelier.salon/account');assert.notEqual(pageKey({id:'home',owner:'a.one'}),pageKey({id:'home',owner:'b.two'}));});
test('page collections use current account kinds and saved lists',()=>{const works=[{id:'film',kind:'films',favorite:true,watchlist:false,metadata:{title:'Film'},localIds:[]},{id:'music',kind:'music',favorite:true,watchlist:true,metadata:{title:'Album'},localIds:[]}];assert.deepEqual(collectionCards('favorites',[],works,['films']).map(c=>c.key),['film']);assert.deepEqual(collectionCards('watchlist',[],works,['films']),[]);assert.deepEqual(collectionCards('watchlist',[],works,['music']).map(c=>c.key),['music']);assert.deepEqual(collectionCards('home',[],works,[]),[]);});

import {feedCards,screenPage} from './ui-pages.ts';
test('media feeds scope resume, recent and saved cards to the account and requested kind',()=>{
 const entries=[{id:'f',kind:'films',name:'Film',libraryId:'l',position:20,duration:100},{id:'m',kind:'music',name:'Music',libraryId:'m',position:10,duration:50}];
 const works=[{id:'f',kind:'films',favorite:true,watchlist:true,metadata:{title:'Film'},localIds:[]},{id:'m',kind:'music',favorite:true,watchlist:true,metadata:{title:'Album'},localIds:[]}];
 for(const source of ['recent','resume','favorites','watchlist']){assert.ok(feedCards({source,view:'home'},entries,works,['films']).every(c=>c.kind==='films'));assert.deepEqual(feedCards({source,view:'music'},entries,works,['films']),[])}
 assert.equal(feedCards({source:'resume',view:'films'},entries,works,['films']).length,1);assert.deepEqual(feedCards({source:'unknown',view:'home'},entries,works,['films']),[]);
});
test('screen selection has no implicit native fallback or ID collision',()=>{const pages=[{id:'films',owner:'a.foo'},{id:'custom',screen:'films',owner:'b.bar'}];assert.equal(screenPage(pages,'films').owner,'b.bar');assert.equal(screenPage(pages,'music'),undefined);assert.equal(screenPage(pages.slice(0,1),'films'),undefined)});
