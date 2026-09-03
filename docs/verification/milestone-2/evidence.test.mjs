import assert from 'node:assert/strict';
import { createHash } from 'node:crypto';
import { readFileSync } from 'node:fs';
import test from 'node:test';

const read = name => readFileSync(new URL(name, import.meta.url));
const hash = bytes => createHash('sha256').update(bytes).digest('hex');

test('issue 46 obtains exact Complete after every Task closes and separate acceptance evidence is ready', () => {
  const bytes = read('46/final-records.ndjson');
  const manifest = JSON.parse(read('46/final-manifest.json'));
  const input = JSON.parse(read('46/operator-input.json'));
  const closure = JSON.parse(read('46/task-closure.json'));
  const baselineManifest = JSON.parse(read('44/baseline-manifest.json'));
  const previous = read('45/attempt-2-records.ndjson').toString().trim().split('\n').map(JSON.parse)[1];
  const records = bytes.toString().trim().split('\n').map(JSON.parse);
  assert.equal(records.length, 2);
  assert.equal(manifest.records, 2);
  assert.equal(manifest.exitCode, 0);
  assert.equal(manifest.signal, null);
  assert.equal(manifest.stdoutBytes, 0);
  assert.equal(manifest.stderrBytes, 0);
  assert.equal(manifest.outputSHA256, hash(bytes));
  assert.equal(manifest.operatorInputSHA256, hash(read('46/operator-input.json')));
  assert.equal(manifest.wrapperSHA256, baselineManifest.wrapperSHA256);
  assert.equal(manifest.binarySHA256, baselineManifest.binarySHA256);
  assert.equal(manifest.observedCodexVersion, baselineManifest.observedCodexVersion);
  const [assessment, report] = records;
  assert.equal(assessment.type, 'assessment');
  assert.equal(report.type, 'processed_report');
  assert.equal(report.classification, 'assessed');
  assert.equal(report.failure, '');
  assert.equal(report.directive, 'await_request');
  assert.equal(report.assessment.outcome, 'complete');
  assert.equal(report.assessment.proposedPlan, null);
  assert.deepEqual(assessment.assessment, report.assessment);
  assert.deepEqual(assessment.target, report.target);
  assert.deepEqual(report.target, previous.target);
  assert.deepEqual(assessment.provenance, report.provenance);
  assert.deepEqual(report.assessment.assessedPlan, report.currentPlan);
  assert.deepEqual(report.currentPlan, previous.currentPlan);
  assert.deepEqual(report.provenance.build, previous.provenance.build);
  assert.equal(input.verifiedRuntimeRevision, report.provenance.build.revision);
  assert.equal(input.verifiedRepositoryRevision, closure.prerequisitePR.mergeCommit);
  assert.equal(report.progress.membershipComplete, true);
  assert.equal(report.progress.tasks.length, 10);
  assert.ok(report.progress.tasks.every(task => task.state === 'closed'));
  assert.deepEqual(report.currentPlan.tasks, report.progress.tasks.map(task => task.name));
  assert.deepEqual(input.acceptanceConditionEvidence.map(item => item.condition), [1, 2, 3, 4, 5]);
  assert.deepEqual(input.acceptanceConditionEvidence.map(item => item.requirement), report.currentPlan.acceptanceConditions);
  assert.ok(input.acceptanceConditionEvidence.every(item => item.facts.length > 0 && item.sources.length > 0));
  assert.equal(closure.before.length, 10);
  assert.equal(closure.after.length, 10);
  assert.deepEqual(closure.before.filter(task => task.state !== 'closed').map(task => task.number), [46]);
  const finalIssue = closure.after.find(task => task.number === 46);
  const orderedTasks = [...closure.after].sort((a, b) => a.number - b.number);
  assert.deepEqual(orderedTasks.map(task => task.number), [44, 45, 46, 50, 51, 52, 53, 54, 55, 56]);
  assert.deepEqual(orderedTasks.map(task => ({ name: task.title, state: task.state })), report.progress.tasks);
  for (const before of closure.before) {
    const after = closure.after.find(task => task.number === before.number);
    assert.equal(after.title, before.title);
    assert.equal(after.state, 'closed');
    assert.equal(after.state_reason, 'completed');
    if (before.number !== 46) {
      assert.deepEqual(after, before);
      assert.ok(Date.parse(after.closed_at) < Date.parse(finalIssue.closed_at));
    }
  }
  const times = [closure.prerequisitePR.mergedAt,
    closure.before.find(task => task.number === 45).closed_at,
    input.verifiedAt, finalIssue.closed_at, manifest.startedAt,
    report.provenance.snapshotAcquisition.startedAt, report.provenance.snapshotAcquisition.completedAt,
    report.provenance.deliveryAcquisition.startedAt, report.provenance.deliveryAcquisition.completedAt,
    manifest.completedAt].map(Date.parse);
  assert.ok(times.every(Number.isFinite));
  assert.deepEqual(times, [...times].sort((a, b) => a - b));
  assert.ok(Date.parse(manifest.startedAt) > Date.parse(finalIssue.closed_at));
});

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

test('issue 45 reacquires facts after two real changes without inventing progress', () => {
  const baseline = read('44/baseline-records.ndjson').toString().trim().split('\n').map(JSON.parse)[1];
  const baselineManifest = JSON.parse(read('44/baseline-manifest.json'));
  const operations = JSON.parse(read('45/operations.json'));
  const before = JSON.parse(read('45/milestone-before.json'));
  const after = JSON.parse(read('45/milestone-after.json'));
  assert.notEqual(before.description, after.description);
  assert.equal(before.description.replace(/^<!-- arcloom-plan:v1\n[A-Za-z0-9+/=\n]+-->\n\n/, ''), after.description);
  assert.equal(hash(before.description), operations.secondChange.descriptionBeforeSHA256);
  assert.equal(hash(after.description), operations.secondChange.descriptionAfterSHA256);
  assert.equal(before.updated_at, operations.secondChange.beforeUpdatedAt);
  assert.equal(after.updated_at, operations.secondChange.afterUpdatedAt);
  assert.ok(Date.parse(after.updated_at) > Date.parse(before.updated_at));
  let previousReport = baseline;
  let previousCompletion = baselineManifest.completedAt;
  for (const [number, changeTime, expectedDifference] of [
    [1, operations.firstChange.closedAt, operations.firstChange.progressDifference],
    [2, operations.secondChange.afterUpdatedAt, operations.secondChange.progressDifference],
  ]) {
    const bytes = read(`45/attempt-${number}-records.ndjson`);
    const manifest = JSON.parse(read(`45/attempt-${number}-manifest.json`));
    const records = bytes.toString().trim().split('\n').map(JSON.parse);
    assert.equal(records.length, 2);
    assert.equal(manifest.records, 2);
    assert.equal(manifest.exitCode, 0);
    assert.equal(manifest.signal, null);
    assert.equal(manifest.outputSHA256, hash(bytes));
    assert.equal(manifest.operatorInputSHA256, hash(read(`45/attempt-${number}-input.json`)));
    assert.equal(manifest.wrapperSHA256, baselineManifest.wrapperSHA256);
    assert.equal(manifest.binarySHA256, baselineManifest.binarySHA256);
    const [assessment, report] = records;
    assert.equal(assessment.type, 'assessment');
    assert.equal(report.type, 'processed_report');
    assert.equal(report.classification, 'assessed');
    assert.equal(report.directive, 'await_request');
    assert.equal(report.failure, '');
    assert.equal(report.assessment.outcome, 'retain');
    assert.deepEqual(assessment.assessment, report.assessment);
    assert.deepEqual(report.assessment.assessedPlan, report.currentPlan);
    assert.deepEqual(report.currentPlan, baseline.currentPlan);
    assert.deepEqual(report.target, baseline.target);
    assert.deepEqual(assessment.target, report.target);
    assert.deepEqual(assessment.provenance, report.provenance);
    assert.deepEqual(report.provenance.build, baseline.provenance.build);
    assert.equal(report.progress.membershipComplete, true);
    assert.equal(report.progress.tasks.length, 10);
    assert.equal(report.progress.tasks.filter(task => task.state === 'closed').length, 8);
    const difference = report.progress.tasks.flatMap((task, index) => {
      const previous = previousReport.progress.tasks[index];
      assert.equal(task.name, previous.name);
      return task.state === previous.state ? [] : [{ name: task.name, before: previous.state, after: task.state }];
    });
    assert.deepEqual(difference, expectedDifference);
    const times = [previousCompletion, changeTime, manifest.startedAt,
      report.provenance.snapshotAcquisition.startedAt, report.provenance.snapshotAcquisition.completedAt,
      report.provenance.deliveryAcquisition.startedAt, report.provenance.deliveryAcquisition.completedAt,
      manifest.completedAt].map(Date.parse);
    assert.ok(times.every(Number.isFinite));
    assert.deepEqual(times, [...times].sort((a, b) => a - b));
    assert.ok(Date.parse(manifest.startedAt) > Date.parse(operations.stateCorrectionExcludedFromProof.reopenedAt));
    previousReport = report;
    previousCompletion = manifest.completedAt;
  }
  assert.deepEqual(operations.secondChange.progressDifference, []);
});
