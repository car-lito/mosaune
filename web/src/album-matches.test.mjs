import test from 'node:test';
import assert from 'node:assert/strict';
import {albumMatchSources,albumMatchRoute} from './album-matches.ts';

test('album sources require actual capability and availability, not a known name',()=>{
 const p={id:'alternative',albumMatches:true,kinds:['music'],enabled:true,configured:true,available:true};
 assert.deepEqual(albumMatchSources([p]),[p]);
 for(const patch of [{albumMatches:false},{kinds:['films']},{enabled:false},{configured:false},{available:false}])assert.deepEqual(albumMatchSources([{...p,...patch}]),[]);
 assert.deepEqual(albumMatchSources([{...p,id:'musicbrainz',albumMatches:undefined}]),[]);
 assert.deepEqual(albumMatchSources([p,{...p,id:'second'}]).map(p=>p.id),['alternative','second']);
 assert.equal(albumMatchRoute('alternative'),'music/matches/alternative');
 assert.equal(albumMatchRoute('unsafe/path'),'music/matches/unsafe%2Fpath');
});
