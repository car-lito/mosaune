import test from 'node:test';
import assert from 'node:assert/strict';
import {setAccount,permitted,canRequest} from './accounts.ts';
test('viewing does not grant playback or edits; new libraries and disabled media remain private',()=>{
 const entry={libraryId:'family',kind:'films'};
 setAccount({admin:false,kinds:['films'],libraries:{family:{view:true,play:false,edit:false,request:false}}});
 assert.equal(permitted(entry,'view'),true);assert.equal(permitted(entry,'play'),false);assert.equal(permitted(entry,'edit'),false);
 assert.equal(permitted({...entry,libraryId:'new'},'view'),false);assert.equal(permitted({...entry,kind:'music'},'view'),false);assert.equal(canRequest('films'),false);
 setAccount(null);assert.equal(permitted(entry,'view'),false);
});
