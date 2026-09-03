import assert from 'node:assert/strict';
import { createHash } from 'node:crypto';
import { readFileSync } from 'node:fs';
import test from 'node:test';

const read = name => readFileSync(new URL(name, import.meta.url));
const hash = bytes => createHash('sha256').update(bytes).digest('hex');

test('issue 44 retains an exact, fresh real baseline and its captured inputs', () => {
  const bytes = read('44/baseline-records.ndjson');
  const manifest = JSON.parse(read('44/baseline-manifest.json'));
  const preflight = JSON.parse(read('44/runtime-preflight.json'));
  const records = bytes.toString().trim().split('\n').map(JSON.parse);
  assert.equal(manifest.records, 2);
  assert.equal(records.length, 2);
  assert.equal(manifest.exitCode, 0);
  assert.equal(manifest.signal, null);
  assert.equal(manifest.outputSHA256, hash(bytes));
  assert.equal(manifest.operatorInputSHA256, hash(read('44/operator-input.json')));
  assert.equal(manifest.wrapperSHA256, hash(read('44/codex-launcher.sh')));
  assert.equal(manifest.binarySHA256, preflight.hostBinarySHA256);
  const [assessment, report] = records;
  assert.equal(assessment.type, 'assessment');
  assert.equal(report.type, 'processed_report');
  assert.equal(report.classification, 'assessed');
  assert.equal(report.failure, '');
  assert.equal(report.directive, 'await_request');
  assert.deepEqual(report.target, { kind: 'github-milestone', key: 'kotokumu/arcloom/milestones/2' });
  assert.deepEqual(assessment.target, report.target);
  assert.deepEqual(assessment.assessment, report.assessment);
  assert.deepEqual(report.assessment.assessedPlan, report.currentPlan);
  assert.equal(report.assessment.outcome, 'retain');
  assert.equal(report.assessment.proposedPlan, null);
  assert.equal(report.progress.membershipComplete, true);
  assert.equal(report.progress.tasks.length, 10);
  assert.equal(report.progress.tasks.filter(task => task.state === 'closed').length, 7);
  assert.deepEqual(report.currentPlan.tasks, report.progress.tasks.map(task => task.name));
  assert.equal(report.currentPlan.acceptanceConditions.length, 5);
  const { provenance } = report;
  assert.deepEqual(assessment.provenance, provenance);
  assert.equal(provenance.build.revision, preflight.hostRevision);
  assert.equal(provenance.build.modified, 'false');
  assert.equal(provenance.codexSupportedVersion, '0.149.1');
  assert.equal(provenance.codexObservedVersion, null);
  assert.equal(manifest.observedCodexVersion, preflight.codex.versionOutput);
  assert.equal(manifest.observedAppServerUserAgent, preflight.codex.appServerUserAgent);
  assert.equal(provenance.model, 'gpt-5.6-sol');
  assert.equal(provenance.effort, 'high');
  assert.equal(provenance.targetURL, 'https://github.com/kotokumu/arcloom/milestone/2');
  assert.equal(provenance.githubAPIVersion, '2022-11-28');
  const times = [manifest.startedAt, provenance.snapshotAcquisition.startedAt,
    provenance.snapshotAcquisition.completedAt, provenance.deliveryAcquisition.startedAt,
    provenance.deliveryAcquisition.completedAt, manifest.completedAt].map(Date.parse);
  assert.ok(times.every(Number.isFinite));
  assert.deepEqual(times, [...times].sort((a, b) => a - b));
});

test('the pre-correction failure is not presented as successful evidence', () => {
  const records = read('44/failed-before-fix.ndjson').toString().trim().split('\n').map(JSON.parse);
  assert.equal(records.length, 1);
  assert.equal(records[0].classification, 'attempt_failure');
  assert.equal(records[0].failure, 'ai_boundary_failure');
  assert.equal(records[0].assessment, null);
  assert.equal(records[0].directive, '');
});
