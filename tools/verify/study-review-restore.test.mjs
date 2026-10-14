import test from 'node:test';
import {execFileSync} from 'node:child_process';
test('14→15 migration audit preserves every old row and allows exactly one valid applied row',()=>{
 execFileSync('python3',['-c',String.raw`
import copy, importlib.util
spec=importlib.util.spec_from_file_location('restore', 'tools/verify/study-review-restore.py')
m=importlib.util.module_from_spec(spec);spec.loader.exec_module(m)
old=[{'id':1,'version_id':0,'is_applied':True,'tstamp':'2026-10-01T00:00:00'},
     {'id':15,'version_id':14,'is_applied':True,'tstamp':'2026-10-13T02:03:04'}]
added={'id':16,'version_id':15,'is_applied':True,'tstamp':'2026-10-14T04:05:06.123456'}
flags=['learning_enabled','correction_enabled','topic_study_enabled','topic_cutover_enabled','managed_knowledge_enabled']
for row in old+[added]: row.update({flag:False for flag in flags})
old[0].update({flag:True for flag in flags})
valid=old+[added]
m.verify_migration_increment(old,valid,15)
def rejects(rows):
    try: m.verify_migration_increment(old,rows,15)
    except ValueError: return
    raise AssertionError('migration mutation was accepted: '+repr(rows))
for field,value in [('id',2),('version_id',13),('is_applied',False),('tstamp','2026-10-01T12:00:00')]:
    changed=copy.deepcopy(valid);changed[0][field]=value;rejects(changed)
for flag in flags:
    changed=copy.deepcopy(valid);changed[0][flag]=False;rejects(changed)
    changed=copy.deepcopy(valid);changed[-1][flag]=True;rejects(changed)
rejects(valid[1:]);rejects(valid+[copy.deepcopy(added)]);rejects(old)
for field,value in [('id',17),('version_id',14),('is_applied',False),('tstamp','invalid')]:
    changed=copy.deepcopy(valid);changed[-1][field]=value;rejects(changed)
changed=copy.deepcopy(valid);changed[-1]['extra']=True;rejects(changed)
`],{cwd:new URL('../../',import.meta.url),stdio:'pipe'});
});
