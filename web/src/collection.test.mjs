import {test} from 'node:test';import assert from 'node:assert/strict';import {groupMedia} from './collection.ts';
const entry=(id,kind,owner,key,n)=>({id,kind,name:id,filename:id,metadata:{series:'Same',album:'Same'},catalogue:{owner,key,title:'Title',subtitle:owner,order:[1,n],sortText:id}});
test('render extension grouping and numeric ordering without media rules',()=>{
 const groups=groupMedia([entry('10','series','a','show',10),entry('2','series','a','show',2),entry('song','music','a','show',1),entry('other','series','b','show',1)]);
 assert.equal(groups.length,3);assert.deepEqual(groups.find(g=>g.entries.length===2).entries.map(e=>e.id),['2','10']);
});
test('no projection or empty group key keeps files individual despite matching metadata',()=>{
 const a=entry('a','music','a','',1),b=entry('b','music','a','',2);delete a.catalogue;assert.equal(groupMedia([a,b]).length,2);
});
