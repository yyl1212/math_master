from pathlib import Path
import runpy
runpy.run_path('/private/tmp/math-master-p5b-case-function-diagnostic.py')
p=Path('/private/tmp/p5b_capacity_diagnostic_test.go');s=p.read_text()
old="RETURNS boolean LANGUAGE sql STABLE AS $$\nWITH evidence AS MATERIALIZED"
new="RETURNS boolean LANGUAGE plpgsql STABLE AS $$\nBEGIN RETURN (WITH evidence AS MATERIALIZED"
assert old in s;s=s.replace(old,new,1)
old="\n$$;`);e!=nil{t.Fatal(e)}";new="\n); END $$;`);e!=nil{t.Fatal(e)}"
assert old in s;s=s.replace(old,new,1);p.write_text(s)
p=Path('/private/tmp/math-master-p5b-case-function-candidate.sql');s=p.read_text();assert 'LANGUAGE sql STABLE AS $$' in s
s=s.replace('LANGUAGE sql STABLE AS $$','LANGUAGE plpgsql STABLE AS $$\nBEGIN RETURN (',1);s=s.replace('\n$$;','\n); END $$;',1);p.write_text(s)
print('Prepared exact STABLE predicate in a PL/pgSQL statement with ordinary prepared-plan reuse, not data caching, only inside the disposable fixture.')
