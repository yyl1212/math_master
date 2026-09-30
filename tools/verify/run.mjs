import { spawn } from 'node:child_process';
import { resolve } from 'node:path';
let timeout = 540000, cwd = process.cwd();
const args = process.argv.slice(2);
while (args.length && args[0] !== '--') {
  const option = args.shift();
  if (option === '--timeout-ms') timeout = Number(args.shift());
  else if (option === '--cwd') cwd = resolve(args.shift() || '');
  else { console.error('Invalid verification option'); process.exit(2); }
}
if (args.shift() !== '--' || !args.length || !Number.isInteger(timeout) || timeout < 1 || timeout > 540000) {
  console.error('Invalid verification command or timeout'); process.exit(2);
}
const grouped = process.platform !== 'win32';
const child = spawn(args[0],args.slice(1),{cwd,stdio:'inherit',detached:grouped});
let timedOut = false, force;
const signal = value => { try { grouped ? process.kill(-child.pid,value) : child.kill(value); } catch (e) { if(e.code !== 'ESRCH') console.error('Unable to terminate verification process'); } };
const timer = setTimeout(() => {
  timedOut = true; console.error('Verification timed out'); signal('SIGTERM');
  force = setTimeout(() => signal('SIGKILL'),5000);
},timeout);
child.on('error',() => { clearTimeout(timer); clearTimeout(force); console.error('Verification command could not start'); process.exitCode=1; });
child.on('close',code => { clearTimeout(timer); clearTimeout(force); process.exitCode=timedOut ? 124 : (code ?? 1); });
for (const sig of ['SIGINT','SIGTERM']) process.on(sig,() => { signal(sig); force ??= setTimeout(() => signal('SIGKILL'),5000); });
