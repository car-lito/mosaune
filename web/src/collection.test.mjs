import { test } from 'node:test';
import assert from 'node:assert/strict';
import { groupMedia } from './collection.ts';
test('group episodes by series, order numerically and keep editions of albums together',()=>{
 const episode=(n)=>({id:`e${n}`,kind:'series',filename:`Show.S01E${n}.mkv`,name:`Episode ${n}`,metadata:{series:'Show',season:1,episode:n}});
 const track=(id,albumArtist,artist,album)=>({id,kind:'music',filename:`${id}.mp3`,name:id,metadata:{albumArtist,artist,album,track:1}});
 const groups=groupMedia([episode(10),episode(2),track('mp3','Artist','Guest','Album'),track('flac','Artist','Artist','Album'),track('other','Someone','Someone','Album')]);
 assert.equal(groups.length,3);
 assert.deepEqual(groups.find(g=>g.kind==='series').entries.map(e=>e.id),['e2','e10']);
 assert.equal(groups.find(g=>g.subtitle==='Artist').entries.length,2);
});
