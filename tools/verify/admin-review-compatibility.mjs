import assert from 'node:assert/strict';
import {wrapTopicCompatibilityReader} from './topic-learning-compatibility.mjs';
import {createHash} from 'node:crypto';
export const exceptionPath='docs/operations/evidence/admin-content-review/compatibility-exceptions.json';
const approvedPaths=[
 'backend/internal/publication/release.go','backend/internal/question/model.go','backend/internal/question/release.go',
 'backend/internal/store/question_release.go','backend/internal/store/question_review.go','backend/internal/store/workflow_release.go','backend/internal/store/workflow_review.go',
];
const newMigration='db/migrations/00009_admin_content_review.sql';
const digest=b=>createHash('sha256').update(b).digest('hex');
export function compareApprovedBytes(baseline,reader) {
 reader=wrapTopicCompatibilityReader(reader);
 const exception=JSON.parse(reader(exceptionPath));
 assert.deepEqual(Object.keys(exception).sort(),['baseCommit','files','newFiles','schemaVersion']);
 assert.equal(exception.schemaVersion,1);
 assert.equal(exception.baseCommit,'610aefff8da3358194a6ec8693f168c0f0cb5a39');
 assert.deepEqual(Object.keys(exception.files).sort(),[...approvedPaths].sort(),'unauthorized compatibility exception');
 assert.deepEqual(Object.keys(exception.newFiles),[newMigration],'unauthorized new protected file');
 for(const path of approvedPaths) {
  const value=exception.files[path];
  assert.deepEqual(Object.keys(value).sort(),['previousSha256','sha256']);
  assert.equal(value.previousSha256,baseline.files[path],path+' previous baseline');
  assert.match(value.sha256,/^[0-9a-f]{64}$/);
 }
 // Historical evidence is never rewritten. Every old file still has an exact expected digest.
 for(const[path,sha]of Object.entries(baseline.files)) {
  assert.equal(digest(reader(path)),exception.files[path]?.sha256??sha,path);
 }
 assert.match(exception.newFiles[newMigration],/^[0-9a-f]{64}$/);
 assert.equal(digest(reader(newMigration)),exception.newFiles[newMigration],newMigration);
}
