"""迁移16发布能力必须在Go、Next、可信HTTPS三层明确就绪。"""
import json
from pathlib import Path
import unittest
from unittest.mock import MagicMock, patch
from test_deploy import module, NEW
import common

class PublishDeploymentTests(unittest.TestCase):
    base={'taxonomy':True,'study':True,'retirement':True,'managed':True,'noteSharing':True,'studyReview':True}

    def test_sixteen_requires_publish_and_seventeen_remains_rejected(self):
        self.assertTrue(module.topicSchemaCompatibility(15,self.base,'topics'))
        self.assertFalse(module.topicSchemaCompatibility(16,self.base,'topics'))
        capabilities={**self.base,'knowledgePublish':True}
        self.assertTrue(module.topicSchemaCompatibility(16,capabilities,'topics'))
        self.assertFalse(module.topicSchemaCompatibility(17,capabilities,'topics'))
        for name in capabilities:
            self.assertFalse(module.topicSchemaCompatibility(16,{**capabilities,name:False},'topics'))
        self.assertFalse(module.topicSchemaCompatibility(16,{**capabilities,'knowledgePublish':1},'topics'))

    def test_health_is_exact_boolean_and_bounded(self):
        for raw in [b'{}',b'[]',b'null',b'x'*2049,b'{"capability":true,"schemaReady":false}',b'{"capability":1,"schemaReady":true}',b'{"capability":true,"schemaReady":true,"body":"private"}']:
            with self.assertRaises((ValueError,TypeError)):module.strict_publish_health(raw)
        self.assertEqual(module.strict_publish_health(b'{"capability":true,"schemaReady":true}'),{'capability':True,'schemaReady':True})

    def test_next_and_public_probe_only_for_sixteen(self):
        with patch.object(module,'expected_migration',return_value=15),patch.object(module,'run_compose',side_effect=AssertionError('old release probed publish')):
            module.verify_publish_next(Path('/private/tmp'),NEW,False)
            module.verify_publish_public(Path('/private/tmp'),NEW)
        with patch.object(module,'expected_migration',return_value=16),patch.object(module,'run_compose',return_value=b'{"capability":true,"schemaReady":true}') as request:
            module.verify_publish_next(Path('/private/tmp'),NEW,False)
            self.assertEqual(request.call_args.args[2][-1],'http://127.0.0.1:3000/api/v3/knowledge-publish/health')
        import urllib.request
        url=common.ORIGIN+'/api/v3/knowledge-publish/health'
        for status,target,body,valid in [(200,url,b'{"capability":true,"schemaReady":true}',True),(404,url,b'{}',False),(200,url+'/redirect',b'{"capability":true,"schemaReady":true}',False),(200,url,b'{"capability":true,"schemaReady":false}',False)]:
            response=MagicMock();response.status=status;response.geturl.return_value=target;response.read.return_value=body;response.__enter__.return_value=response
            with patch.object(module,'expected_migration',return_value=16),patch.object(urllib.request,'urlopen',return_value=response):
                if valid:
                    module.verify_publish_public(Path('/private/tmp'),NEW);response.read.assert_called_once_with(2049)
                else:
                    with self.assertRaises(module.DeployError):module.verify_publish_public(Path('/private/tmp'),NEW)

    def test_missing_go_publish_blocks_sixteen(self):
        health={**{k:v for k,v in self.base.items() if k not in ['noteSharing','studyReview']},'schemaReady':True,'topicsMode':True,'managedMode':True}
        with patch.object(module,'topic_database_state',return_value={'requiredMigration':16,'mode':'topics','managedMode':True}),patch.object(module,'runtime_topic_health',return_value=health),patch.object(module,'runtime_note_sharing_health',return_value={'capability':True,'schemaReady':True}),patch.object(module,'runtime_study_review_health',return_value={'capability':True,'schemaReady':True}),patch.object(module,'runtime_publish_health',side_effect=module.DeployError('publish-readiness-unavailable')):
            with self.assertRaises(module.DeployError):module.verify_topic_runtime(Path('/private/tmp'),NEW,False,'container')

    def test_publish_probes_precede_ready_and_current(self):
        order=[];state={'bootstrapVerified':True,'runningRevision':NEW,'current':NEW}
        def compose(root,revision,args,sudo=False,failure=''):
            order.append('web' if args[:2]==['up','-d'] and 'web' in args else 'api' if args[:2]==['up','-d'] and 'api' in args else 'other');return b''
        with patch.object(module,'database_container',return_value='container'),patch.object(module,'stop_writes'),patch.object(module.snapshot,'capture'),patch.object(module,'run_compose',side_effect=compose),patch.object(module,'verify_topic_runtime',side_effect=lambda *a:order.append('go')),patch.object(module,'verify_note_sharing_next'),patch.object(module,'verify_study_review_next'),patch.object(module,'verify_publish_next',side_effect=lambda *a:order.append('publish-next')),patch.object(module,'save_state',side_effect=lambda *a:order.append('ready')):
            module.start(Path('/private/tmp'),NEW,False,state,None)
        self.assertLess(order.index('api'),order.index('go'));self.assertLess(order.index('web'),order.index('publish-next'));self.assertLess(order.index('publish-next'),order.index('ready'))
        order=[]
        with patch.object(module,'database_container',return_value='container'),patch.object(module,'verify_topic_runtime',side_effect=lambda *a:order.append('go')),patch.object(module,'run_compose',side_effect=lambda *a:order.append('gateway')),patch.object(module,'trusted_check',side_effect=lambda *a:order.append('https')),patch.object(module,'verify_note_sharing_public'),patch.object(module,'verify_study_review_public'),patch.object(module,'verify_publish_public',side_effect=lambda *a:order.append('publish-public')),patch.object(module,'publish_current',side_effect=lambda *a:order.append('current')):
            module.activate(Path('/private/tmp'),NEW,False,state,'production')
        self.assertLess(order.index('https'),order.index('publish-public'));self.assertLess(order.index('publish-public'),order.index('current'))
