#!/usr/bin/env python3
"""默认系统 CA 的 HTTPS、公开资源、匿名权限及有界只读验收。"""
import argparse
from concurrent.futures import ThreadPoolExecutor
from datetime import datetime, timezone
from html import unescape
from http.cookies import SimpleCookie
import ipaddress
import json
import math
import os
from pathlib import Path
import re
import socket
import ssl
import sys
import time
from urllib.error import HTTPError
from urllib.parse import urljoin,urlsplit
from urllib.request import build_opener,HTTPRedirectHandler,HTTPSHandler,Request

import common

DRAFT='/api/v1/content/drafts/904d550b-226b-48c0-98e0-610af339826c'
PREVIEW='/editor/drafts/904d550b-226b-48c0-98e0-610af339826c/preview'
HEADERS={'Accept':'application/json, text/html;q=0.9, */*;q=0.8','X-Requested-With':'MathMaster','User-Agent':'MathMasterDeploymentCheck/1'}


class VerifyError(ValueError):pass


class NoRedirect(HTTPRedirectHandler):
    def redirect_request(self,req,fp,code,msg,headers,newurl):return None


def request(url: str,*,redirect: bool=False) -> dict:
    handlers=[HTTPSHandler(context=ssl.create_default_context())]
    if not redirect:handlers.append(NoRedirect())
    opener=build_opener(*handlers)
    try:
        response=opener.open(Request(url,headers=HEADERS),timeout=5)
    except HTTPError as error:
        response=error
    with response:
        body=response.read(4*1024*1024+1)
        if len(body)>4*1024*1024:raise VerifyError('response-too-large')
        return {'status':response.status,'headers':response.headers,'body':body}


def tls(origin: str) -> dict:
    parsed=urlsplit(origin)
    if parsed.scheme!='https' or not parsed.hostname or parsed.username or parsed.password:
        raise VerifyError('unsafe-tls-origin')
    address=ipaddress.ip_address(parsed.hostname)
    context=ssl.create_default_context()
    with socket.create_connection((parsed.hostname,parsed.port or 443),timeout=5) as transport:
        with context.wrap_socket(transport,server_hostname=parsed.hostname) as connection:
            certificate=connection.getpeercert()
    addresses=[value for kind,value in certificate.get('subjectAltName',()) if kind=='IP Address']
    if str(address) not in addresses:raise VerifyError('missing-ip-san')
    expires=datetime.fromtimestamp(ssl.cert_time_to_seconds(certificate['notAfter']),timezone.utc)
    if expires<=datetime.now(timezone.utc):raise VerifyError('expired-certificate')
    issuer=' / '.join(value for group in certificate.get('issuer',()) for key,value in group if key in ['organizationName','commonName'])
    return {'trusted':True,'ipSAN':addresses,'issuer':issuer,'notAfter':expires.isoformat()}


def asset_url(base: str,value: str,origin: str) -> str:
    resolved=urljoin(base,unescape(value))
    parsed=urlsplit(resolved)
    if parsed.scheme+'://'+parsed.netloc!=origin or parsed.username or parsed.password:
        raise VerifyError('unexpected-resource-origin')
    return resolved


def read_ok(url: str) -> dict:
    response=request(url)
    if response['status']!=200:raise VerifyError('public-resource-unavailable')
    return response


def check(origin: str,requests: int,concurrency: int) -> dict:
    if origin!=common.ORIGIN or not 1<=requests<=100 or not 1<=concurrency<=4:
        raise VerifyError('invalid-verification-input')
    certificate=tls(origin)
    redirect=request('http://43.135.142.53/login')
    if redirect['status'] not in [301,302,307,308] or redirect['headers'].get('Location')!=origin+'/login':
        raise VerifyError('http-redirect-invalid')
    login=read_ok(origin+'/login')
    read_ok(origin+'/register')
    context=read_ok(origin+'/api/v1/auth/context')
    cookies=SimpleCookie()
    for value in context['headers'].get_all('Set-Cookie',[]):cookies.load(value)
    cookie=cookies.get('__Host-mm_preauth')
    if cookie is None or not cookie['secure'] or not cookie['httponly'] or cookie['path']!='/' or cookie['domain'] or cookie['samesite'].lower()!='lax':
        raise VerifyError('production-cookie-invalid')
    if request(origin+DRAFT)['status']!=401:raise VerifyError('anonymous-draft-accessible')
    preview=read_ok(origin+PREVIEW)['body'].decode('utf-8')
    if 'Sign in to view your account' not in preview or 'href="/login"' not in preview or 'PRIVATE DRAFT PREVIEW' in preview:
        raise VerifyError('anonymous-preview-invalid')
    html=login['body'].decode('utf-8')
    stylesheets={asset_url(origin+'/login',value,origin) for value in re.findall(r'href="([^\"]+\.css(?:\?[^\"]*)?)"',html)}
    if not stylesheets:raise VerifyError('stylesheets-missing')
    fonts=set()
    for stylesheet in stylesheets:
        text=read_ok(stylesheet)['body'].decode('utf-8')
        for value in re.findall(r'url\(["\']?([^\s)"\']+)["\']?\)',text):
            if re.search(r'KaTeX[^/]*\.woff2(?:\?|$)',value,re.I):fonts.add(asset_url(stylesheet,value,origin))
    if not fonts:raise VerifyError('formula-fonts-missing')
    for font in fonts:
        if not read_ok(font)['body']:raise VerifyError('formula-font-empty')
    paths=['/','/knowledge','/login','/register']
    def sample(index: int) -> tuple[bool,float]:
        started=time.perf_counter()
        try:ok=request(origin+paths[index%len(paths)])['status']==200
        except (OSError,ValueError):ok=False
        return ok,(time.perf_counter()-started)*1000
    with ThreadPoolExecutor(max_workers=concurrency) as pool:
        samples=list(pool.map(sample,range(requests)))
    elapsed=sorted(value for _,value in samples)
    failures=sum(not ok for ok,_ in samples)
    percentile=lambda p: round(elapsed[max(0,math.ceil(len(elapsed)*p)-1)],3)
    return {'schemaVersion':1,'ok':failures==0,'origin':origin,'checkedAt':datetime.now(timezone.utc).isoformat(),'tls':certificate,
            'httpRedirect':True,'publicPages':True,'anonymousDraftDenied':True,'anonymousPreviewSignIn':True,'productionCookieAttributes':True,
            'stylesheets':len(stylesheets),'katexFonts':len(fonts),'load':{'requests':requests,'concurrency':concurrency,'failures':failures,'p50Ms':percentile(.5),'p95Ms':percentile(.95),'maxMs':round(max(elapsed),3)}}


def main(argv: list[str] | None=None) -> int:
    parser=argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--origin',required=True);parser.add_argument('--out',type=Path,required=True)
    parser.add_argument('--requests',type=int,default=40);parser.add_argument('--concurrency',type=int,default=2)
    parser.add_argument('--revision')
    args=parser.parse_args(argv)
    try:
        target=common.safe_path(args.out)
        if target.exists():raise VerifyError('evidence-already-exists')
        if args.revision:common.validate_revision(args.revision)
        result=check(args.origin,args.requests,args.concurrency)
        if args.revision:result['revision']=args.revision
        target.parent.mkdir(mode=0o700,parents=True,exist_ok=True)
        descriptor=os.open(target,os.O_WRONLY|os.O_CREAT|os.O_EXCL|os.O_NOFOLLOW,0o600)
        with os.fdopen(descriptor,'w') as stream:json.dump(result,stream,indent=2);stream.write('\n')
        print(json.dumps({'ok':result['ok'],'requests':result['load']['requests'],'failures':result['load']['failures']}))
        return 0 if result['ok'] else 1
    except (OSError,ValueError,KeyError) as error:
        stage=str(error) if re.fullmatch(r'[a-z-]{1,64}',str(error)) else 'https-verification-failed'
        print(json.dumps({'ok':False,'stage':stage}),file=sys.stderr);return 1


if __name__=='__main__':sys.exit(main())
