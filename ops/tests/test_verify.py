"""可信 TLS、匿名草稿与资源验收；证据不能携带会话材料。"""
from email.message import Message
import http.server
import importlib.util
import json
from pathlib import Path
import ssl
import subprocess
import sys
import tempfile
import threading
import unittest
from unittest.mock import patch

OPS=Path(__file__).resolve().parents[1]
sys.path.insert(0,str(OPS))
module=None
if (OPS/'verify-deployment.py').exists():
    spec=importlib.util.spec_from_file_location('deployment_verifier',OPS/'verify-deployment.py')
    module=importlib.util.module_from_spec(spec);spec.loader.exec_module(module)
ORIGIN='https://43.135.142.53'
DRAFT='/api/v1/content/drafts/904d550b-226b-48c0-98e0-610af339826c'
TLS={'trusted':True,'ipSAN':['43.135.142.53'],'issuer':'TLS fixture','notAfter':'2026-10-11T00:00:00+00:00'}


def response(status,body=b'',**headers):
    values=Message()
    for name,value in headers.items():values[name.replace('_','-')]=value
    return {'status':status,'headers':values,'body':body}


class VerifyTests(unittest.TestCase):
    def implementation(self):
        self.assertIsNotNone(module,'deployment verifier is missing')
        return module

    def transport(self,url,**kwargs):
        if url.endswith('/readyz'):return response(200,json.dumps({'status':'ready','topic':{'taxonomy':True,'study':True,'retirement':True,'schemaReady':True,'topicsMode':False}}).encode())
        if url.startswith('http://'):
            return response(308,Location=ORIGIN+'/login')
        if url.endswith('/api/v1/auth/context'):
            return response(200,b'{"data":{"user":null,"csrfToken":"csrf-must-not-be-recorded"}}',Set_Cookie='__Host-mm_preauth=session-must-not-be-recorded; Path=/; Secure; HttpOnly; SameSite=Lax; Max-Age=600')
        if url.endswith(DRAFT):return response(401,b'{"error":{"code":"AUTH_REQUIRED"}}')
        if url.endswith('/preview'):
            return response(200,b'<h1>Sign in to view your account</h1><a href="/login">Sign in</a>')
        if url.endswith('.css'):
            return response(200,b'@font-face{font-family:KaTeX_Main;src:url("../media/KaTeX_Main-Regular.woff2")}')
        if url.endswith('.woff2'):return response(200,b'font-fixture')
        return response(200,b'<html><link rel="stylesheet" href="/_next/static/chunks/site.css"><body>public page</body></html>')

    def test_untrusted_tls_fails(self):
        verifier=self.implementation()
        with tempfile.TemporaryDirectory() as directory:
            root=Path(directory);config=root/'certificate.conf';certificate=root/'cert.pem';key=root/'key.pem'
            config.write_text('[req]\ndistinguished_name=dn\nx509_extensions=ext\nprompt=no\n[dn]\nCN=127.0.0.2\n[ext]\nsubjectAltName=IP:127.0.0.2\nbasicConstraints=CA:TRUE\n')
            result=subprocess.run(['openssl','req','-x509','-newkey','rsa:2048','-nodes','-keyout',str(key),'-out',str(certificate),'-days','1','-config',str(config)],capture_output=True)
            self.assertEqual(result.returncode,0,'TLS fixture generation failed')
            key.chmod(0o600)
            server=http.server.ThreadingHTTPServer(('127.0.0.1',0),http.server.BaseHTTPRequestHandler)
            context=ssl.SSLContext(ssl.PROTOCOL_TLS_SERVER);context.load_cert_chain(certificate,key)
            server.socket=context.wrap_socket(server.socket,server_side=True)
            thread=threading.Thread(target=server.serve_forever,daemon=True);thread.start()
            try:
                origin='https://127.0.0.1:'+str(server.server_port)
                with self.assertRaises(ssl.SSLError):verifier.tls(origin)
                trusted_fixture=ssl.create_default_context(cafile=str(certificate))
                with patch.object(verifier.ssl,'create_default_context',return_value=trusted_fixture):
                    with self.assertRaises(ssl.SSLError):verifier.tls(origin)
            finally:
                server.shutdown();server.server_close();thread.join(timeout=5)

    def test_anonymous_draft_is_private(self):
        verifier=self.implementation()
        with patch.object(verifier,'tls',return_value=TLS),patch.object(verifier,'request',side_effect=self.transport):
            self.assertTrue(verifier.check(ORIGIN,2,1)['ok'])
        def leaked(url,**kwargs):
            if url.endswith(DRAFT):return response(200,b'private mathematics')
            return self.transport(url,**kwargs)
        with patch.object(verifier,'tls',return_value=TLS),patch.object(verifier,'request',side_effect=leaked):
            with self.assertRaises(verifier.VerifyError):verifier.check(ORIGIN,2,1)

    def test_missing_static_asset_fails(self):
        verifier=self.implementation()
        def missing(url,**kwargs):
            if url.endswith('.woff2'):return response(404)
            return self.transport(url,**kwargs)
        with patch.object(verifier,'tls',return_value=TLS),patch.object(verifier,'request',side_effect=missing):
            with self.assertRaises(verifier.VerifyError):verifier.check(ORIGIN,2,1)

    def test_evidence_omits_session_material(self):
        verifier=self.implementation()
        with patch.object(verifier,'tls',return_value=TLS),patch.object(verifier,'request',side_effect=self.transport):
            evidence=verifier.check(ORIGIN,4,2)
        encoded=json.dumps(evidence)
        self.assertNotIn('session-must-not-be-recorded',encoded)
        self.assertNotIn('csrf-must-not-be-recorded',encoded)
        self.assertNotIn('Set-Cookie',encoded)
        self.assertNotIn('public page',encoded)
        self.assertEqual(evidence['load']['requests'],4)
        self.assertEqual(evidence['load']['failures'],0)


if __name__=='__main__':unittest.main()

class TopicReadinessTests(unittest.TestCase):
    def test_topic_readiness_contains_only_safe_booleans(self):
        good={'taxonomy':True,'study':True,'retirement':True,'schemaReady':True,'topicsMode':True}
        self.assertEqual(module.verify_topic_readiness({'status':'ready','topic':good}),good)
        for topic in [{**good,'privateNote':'must not be evidence'},{**good,'retirement':False},{**good,'schemaReady':False},{**good,'study':'true'}]:
            with self.assertRaises(module.VerifyError):module.verify_topic_readiness({'status':'ready','topic':topic})

class ManagedReadinessTests(unittest.TestCase):
    def test_retired_preview_accepts_only_exact_safe_redirect_transports(self):
        module.verify_retired_preview(response(307,Location='/account'))
        for delay in ['0','1']:
            body=('<html><meta id="__next-page-redirect" http-equiv="refresh" content="'+delay+';url=/account"/></html>').encode()
            module.verify_retired_preview(response(200,body,Content_Type='text/html; charset=utf-8'))
        for bad in [response(200,b'private mathematics'),response(307,Location='https://other.example/account'),response(200,b'<meta id="__next-page-redirect" http-equiv="refresh" content="1;url=https://other.example/account"/>'),response(200,b'<meta id="__next-page-redirect" http-equiv="refresh" content="5;url=/account"/>'),response(200,b'<meta id="__next-page-redirect" http-equiv="refresh" content="1;url=/account"/> PRIVATE DRAFT PREVIEW'),response(200,b'<meta id="__next-page-redirect" http-equiv="refresh" content="1;url=/account"/>'*2)]:
            if bad['status']==200:bad['headers']['Content-Type']='text/html; charset=utf-8'
            with self.assertRaises(module.VerifyError):module.verify_retired_preview(bad)
        marker=b'<meta id="__next-page-redirect" http-equiv="refresh" content="1;url=/account"/>'
        for body in [b'<base href="https://other.example/">'+marker,b'<template>'+marker+b'</template>',b'<noscript>'+marker+b'</noscript>']:
            with self.assertRaises(module.VerifyError):module.verify_retired_preview(response(200,body,Content_Type='text/html; charset=utf-8'))
    def test_current_content_readiness_is_strict_and_safe(self):
        good={'capability':True,'schemaReady':True,'managedMode':True}
        self.assertEqual(module.verify_content_readiness({'status':'ready','content':good}),good)
        for bad in [{**good,'schemaReady':False},{**good,'capability':False},{**good,'privateNotes':'secret'},{**good,'managedMode':'true'}]:
            with self.assertRaises(module.VerifyError):module.verify_content_readiness({'status':'ready','content':bad})
