import test from 'node:test';
import assert from 'node:assert/strict';
import {catalogCandidates} from './repository-catalog.ts';

const entry={id:'atelier.ambre',name:'Ambre',publisher:'Mosaune',description:'Papier et ambre',version:'2.0.0',kind:'theme',minCore:'0.0.15',path:'ambre.zip',sha256:'a'.repeat(64)};
const repository=(id,packages=[entry])=>({id,url:'https://github.com/a/'+id,ref:'',updatedAt:1,bundled:id==='bundled',official:id==='official',index:{name:id,packages}});
const list=[repository('bundled'),repository('official')];

test('identical bundled and official packages produce one card, regardless of order',()=>{
 for(const sources of [list,[...list].reverse()]){
  const cards=catalogCandidates(sources,'','','');
  assert.equal(cards.length,1);assert.equal(cards[0].repo.id,'official');
 }
 assert.equal(catalogCandidates([list[0]],'','','')[0].repo.id,'bundled');
});
test('source filter keeps the selected source, search and kind filters remain effective',()=>{
 assert.equal(catalogCandidates(list,'ambre','bundled','theme')[0].repo.id,'bundled');
 assert.equal(catalogCandidates(list,'ambre','official','theme')[0].repo.id,'official');
 assert.equal(catalogCandidates(list,'missing','','').length,0);
 assert.equal(catalogCandidates(list,'','','provider').length,0);
});
test('same ID or version cannot hide a different archive or manifest',()=>{
 for(const change of [{sha256:'b'.repeat(64)},{publisher:'Another'},{description:'Different'},{minCore:'9.0.0'},{name:'Different'}]){
  const third=repository('community',[{...entry,...change}]);
  assert.equal(catalogCandidates([...list,third],'','','').length,2);
 }
 const mirror=repository('community',[{...entry,path:'other/ambre.zip'}]);
 assert.equal(catalogCandidates([...list,mirror],'','','').length,1);
});
test('newer releases are not hidden by deduplication and older versions remain in their source',()=>{
 const update={...entry,version:'2.0.1',sha256:'b'.repeat(64)};
 const source=repository('official',[entry,update]);
 const cards=catalogCandidates([list[0],source],'','','');
 assert.equal(cards.length,2);assert.deepEqual(cards.map(c=>c.entry.version),['2.0.0','2.0.1']);
 assert.equal(source.index.packages.length,2);
});
