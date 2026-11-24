import test from 'node:test';
import assert from 'node:assert/strict';
import {artistResolvers,artistResolutionRoute,personProfileRoute} from './people.ts';

test('artist sources come from implemented capabilities rather than provider names',()=>{
 const source={id:'other',profiles:true,resolveArtists:true,kinds:['music'],enabled:true,available:true,configured:true};
 assert.deepEqual(artistResolvers([source]),[source]);
 for(const patch of [{profiles:false},{resolveArtists:false},{kinds:['films']},{enabled:false},{available:false},{configured:false}])assert.deepEqual(artistResolvers([{...source,...patch}]),[]);
 assert.deepEqual(artistResolvers([{id:'musicbrainz',kinds:['music'],enabled:true,configured:true}]),[]);
 const second={...source,id:'second'};assert.deepEqual(artistResolvers([source,second]),[source,second]);
 assert.deepEqual(artistResolvers([]),[]);
});
test('resolution and profile links keep the selected provider and escape identifiers',()=>{
 assert.equal(artistResolutionRoute('other'),' /api/v1/people/other/resolve'.trim());
 assert.equal(personProfileRoute('other','artist/a?b',2),'/api/v1/people/other/artist%2Fa%3Fb?page=2');
});
