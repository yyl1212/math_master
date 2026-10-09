"""准备与迁入独立、有界、使用准确受保护凭证。"""
import importlib.util
from pathlib import Path
import sys,tempfile,unittest
OPS=Path(__file__).resolve().parents[1];sys.path.insert(0,str(OPS))
class KnowledgeCutoverTests(unittest.TestCase):
 def module(self):
  spec=importlib.util.spec_from_file_location('knowledge_cutover',OPS/'knowledge-cutover.py');m=importlib.util.module_from_spec(spec);spec.loader.exec_module(m);return m
 def test_steps_are_separate_and_at_most_five_minutes(self):
  m=self.module();self.assertEqual(m.STEP_SECONDS,300);self.assertEqual(m.OPERATIONS,{'prepare','activate','clean-old'});self.assertNotIn('all',m.OPERATIONS)
 def test_private_receipt_rejects_wrong_permissions_or_replacement(self):
  m=self.module()
  with tempfile.TemporaryDirectory() as d:
   p=Path(d).resolve()/'receipt.json';m.write_receipt(p,{'verified':True});self.assertEqual(p.stat().st_mode&0o777,0o600)
   with self.assertRaises(ValueError):m.write_receipt(p,{'verified':False})

 def test_preparation_accepts_current_fractional_timestamp_and_rejects_stale_backup(self):
  from datetime import datetime,timezone,timedelta
  from unittest.mock import patch
  m=self.module();now=datetime.now(timezone.utc)
  data={'createdAt':now.isoformat(),'dumpSha256':'a'*64,'database':{'name':'math_master_preview'}}
  with tempfile.TemporaryDirectory() as d:
   root=Path(d).resolve();backup=root/'backup';copy=root/'copy';backup.mkdir();copy.mkdir()
   with patch.object(m.snapshot,'load_backup',return_value=data),patch.object(m.snapshot,'drill',return_value={'ok':True}),patch.object(m.snapshot,'digest_file',return_value='b'*64):
    self.assertTrue(m.prepare(backup,copy,'verified-other-host')['restoreVerified'])
    data['createdAt']=(now-timedelta(hours=1)).isoformat()
    with self.assertRaises(ValueError):m.prepare(backup,copy,'verified-other-host')
