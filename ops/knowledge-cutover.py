#!/usr/bin/env python3
"""当前知识切换：准备与迁入独立，每次操作最多五分钟。"""
import argparse,importlib.util,json,os,signal,socket,stat,subprocess,sys,time
from datetime import datetime,timezone
from pathlib import Path
import common
STEP_SECONDS=300
OPERATIONS={'prepare','activate','clean-old'}
spec=importlib.util.spec_from_file_location('knowledge_snapshot',Path(__file__).with_name('database-snapshot.py'))
snapshot=importlib.util.module_from_spec(spec);spec.loader.exec_module(snapshot)
def write_receipt(path,value):
 path=common.safe_path(path)
 if path.exists():raise ValueError('receipt-already-exists')
 with os.fdopen(os.open(path,os.O_WRONLY|os.O_CREAT|os.O_EXCL|os.O_NOFOLLOW,0o600),'w') as f:json.dump(value,f,ensure_ascii=False,sort_keys=True);f.write('\n')
def private_input(path):
 path=common.safe_path(path);info=path.lstat()
 if not stat.S_ISREG(info.st_mode) or stat.S_IMODE(info.st_mode)!=0o600 or info.st_uid!=os.geteuid() or info.st_size>2*1024*1024:raise ValueError('unsafe-input')
 return path

def prepare(backup,offsite_copy,offsite_host,sudo=False):
 backup=common.safe_path(backup);copy=common.safe_path(offsite_copy)
 if backup==copy or not offsite_host or offsite_host==socket.gethostname():raise ValueError('offsite-copy-required')
 data=snapshot.load_backup(backup);other=snapshot.load_backup(copy)
 if data!=other:raise ValueError('offsite-copy-mismatch')
 created=datetime.fromisoformat(data['createdAt']);now=datetime.now(timezone.utc)
 if created.tzinfo is None or not 0<=(now-created).total_seconds()<=1800:raise ValueError('backup-not-current')
 drill=snapshot.drill(backup,sudo)
 if drill.get('ok') is not True:raise ValueError('restore-verification-failed')
 return {'recordSha':snapshot.digest_file(backup/'manifest.json'),'dumpSha':data['dumpSha256'],'database':data['database']['name'],'backupCreatedAt':data['createdAt'],'checkedAt':datetime.now(timezone.utc).isoformat(),'restoreVerified':True,'offsiteVerified':True,'offsiteHost':offsite_host}

def apply(operation,binary,input_path,backup,verification):
 if operation not in {'activate','clean-old'}:raise ValueError('invalid-step')
 binary=common.safe_path(binary);info=binary.lstat()
 if not stat.S_ISREG(info.st_mode) or not os.access(binary,os.X_OK):raise ValueError('maintenance-binary-unavailable')
 result=subprocess.run([str(binary),operation,'--input',str(private_input(input_path)),'--backup',str(common.safe_path(backup)),'--verification',str(private_input(verification))],capture_output=True,timeout=STEP_SECONDS-5)
 if result.returncode or len(result.stdout)>2*1024*1024:raise ValueError('maintenance-step-refused')
 return json.loads(result.stdout)

def main(argv=None):
 p=argparse.ArgumentParser(description=__doc__);p.add_argument('operation',choices=sorted(OPERATIONS));p.add_argument('--backup',type=Path,required=True);p.add_argument('--out',type=Path,required=True);p.add_argument('--offsite-copy',type=Path);p.add_argument('--offsite-host');p.add_argument('--sudo',action='store_true');p.add_argument('--binary',type=Path);p.add_argument('--input',type=Path);p.add_argument('--verification',type=Path);a=p.parse_args(argv)
 def timed_out(*_):raise ValueError('step-timeout')
 previous=signal.signal(signal.SIGALRM,timed_out);signal.setitimer(signal.ITIMER_REAL,STEP_SECONDS)
 started=time.monotonic()
 try:
  if a.operation=='prepare':
   if not a.offsite_copy or not a.offsite_host or a.binary or a.input or a.verification:raise ValueError('invalid-step')
   result=prepare(a.backup,a.offsite_copy,a.offsite_host,a.sudo)
  else:
   if not a.binary or not a.input or not a.verification or a.offsite_copy or a.offsite_host or a.sudo:raise ValueError('invalid-step')
   result=apply(a.operation,a.binary,a.input,a.backup,a.verification)
  write_receipt(a.out,result);print(json.dumps({'ok':True,'step':a.operation,'elapsedSeconds':round(time.monotonic()-started,2)}));return 0
 except (OSError,ValueError,subprocess.SubprocessError):print(json.dumps({'ok':False,'step':a.operation,'code':'STEP_REFUSED'}),file=sys.stderr);return 1
 finally:signal.setitimer(signal.ITIMER_REAL,0);signal.signal(signal.SIGALRM,previous)
if __name__=='__main__':sys.exit(main())
