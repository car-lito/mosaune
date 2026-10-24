import test from 'node:test';
import assert from 'node:assert/strict';
import {catalogCandidates} from './repository-catalog.ts';

test('recognition catalogues expose the latest release with a distinct family and source',()=>{
 const e={id:'example.film-names',name:'Film names',publisher:'Author',description:'Recognizer',version:'1.0.0',kind:'recognition',minCore:'0.0.31',path:'packages/one.zip',sha256:'a'.repeat(64)};
 const update={...e,version:'1.1.0',path:'packages/two.zip',sha256:'b'.repeat(64)};
 const repo={id:'community',url:'https://github.com/author/repo',ref:'',updatedAt:1,bundled:false,official:false,index:{name:'Community',packages:[e,update]}};
 const cards=catalogCandidates([repo],'film','','recognition');
 assert.equal(cards.length,1);assert.equal(cards[0].entry.version,'1.1.0');assert.equal(cards[0].repo.id,'community');
 assert.equal(catalogCandidates([repo],'','','provider').length,0);
 assert.equal(repo.index.packages.length,2);
});
