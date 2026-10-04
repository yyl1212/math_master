from pathlib import Path
import json,subprocess
r=Path('/Users/wiw/.codex/worktrees/p5-feedback-design/math_master')
head='424a6a279f7fc1fef6ec10abe1cc918743cb425d'
a=r/'backend/internal/store/learning_fixture_test.go';b=r/'backend/internal/store/learning_qualification_test.go'
s=subprocess.check_output(['git','show',head+':backend/internal/store/learning_fixture_test.go'],cwd=r,text=True)
old='now := time.Now().UTC()';assert s.count(old)==1;s=s.replace(old,'now := time.Now().UTC().Add(5 * time.Second)')
f=Path('/private/tmp/p5b-clock-skew-fixture.go');f.write_text(s)
s=subprocess.check_output(['git','show',head+':backend/internal/store/learning_qualification_test.go'],cwd=r,text=True)
key='\tif _, e = tx.Exec(`UPDATE assessment_attempts SET state=\'submitted\'';pos=s.index(key)
s=s[:pos]+'''\tvar created, databaseNow time.Time
\tvar definition string
\tif e = tx.QueryRow(`SELECT created_at,clock_timestamp(),(SELECT pg_get_constraintdef(oid) FROM pg_constraint WHERE conrelid='public.assessment_attempts'::regclass AND conname='assessment_attempts_check3') FROM assessment_attempts WHERE id=$1`,id).Scan(&created,&databaseNow,&definition);e!=nil{f.t.Fatal(e)}
\tf.t.Logf("CLOCK_SOURCE_DIAGNOSTIC constraint=%s created=%s dbNow=%s terminal=%s ahead=%s",definition,created.UTC(),databaseNow.UTC(),now.UTC(),created.Sub(now))
'''+s[pos:]
g=Path('/private/tmp/p5b-clock-skew-qualification.go');g.write_text(s)
Path('/private/tmp/math-master-p5b-clock-skew-overlay.json').write_text(json.dumps({'Replace':{str(a):str(f),str(b):str(g)}}))
print('Prepared disposable original-424 fixture with controlled host clock +5 seconds; repository remains unchanged.')
