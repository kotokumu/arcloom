#!/usr/bin/env node

import { spawnSync } from 'node:child_process';
import fs from 'node:fs';
import path from 'node:path';
import process from 'node:process';

const SUPPORTED_OPENSPEC_VERSION = '1.10.0';
const CHANGE_NAME = /^[a-z0-9][a-z0-9-]*$/;
const PATH_SEGMENT = '[a-z0-9]+(?:-[a-z0-9]+)*';
const CAPABILITY_PATH = new RegExp(`^${PATH_SEGMENT}(?:/${PATH_SEGMENT})*$`);
const ALLOWED_DELTA_SECTIONS = [
  /^Purpose$/i,
  /^(ADDED|MODIFIED|REMOVED|RENAMED) Requirements$/i,
];
const MODEL_SECTION = 'Main Spec Conceptual Model Replacements';

function unnumberedTitle(title) {
  return title.replace(/^\d+\.\s+/, '');
}

function abort(message) {
  throw new Error(message);
}

function findProjectRoot(start) {
  let current = path.resolve(start);
  while (true) {
    if (fs.existsSync(path.join(current, 'openspec', 'config.yaml'))) return current;
    const parent = path.dirname(current);
    if (parent === current) abort('Could not find openspec/config.yaml from the current directory.');
    current = parent;
  }
}

function runOpenSpec(root, args, { print = false } = {}) {
  const result = spawnSync('openspec', args, {
    cwd: root,
    encoding: 'utf8',
    stdio: print ? 'inherit' : 'pipe',
  });
  if (result.error) abort(`Failed to execute openspec: ${result.error.message}`);
  if (result.status !== 0) {
    const detail = [result.stdout, result.stderr].filter(Boolean).join('\n').trim();
    abort(`openspec ${args.join(' ')} failed${detail ? `:\n${detail}` : '.'}`);
  }
  return (result.stdout ?? '').trim();
}

function structuralLines(content) {
  const lines = content.replace(/\r\n?/g, '\n').split('\n');
  const visible = [];
  let fence = null;
  let inComment = false;

  for (let index = 0; index < lines.length; index += 1) {
    const line = lines[index];

    if (fence) {
      if (new RegExp(`^ {0,3}${fence.char}{${fence.length},}\\s*$`).test(line)) fence = null;
      visible.push(null);
      continue;
    }

    let remainder = line;
    let markdown = '';
    while (remainder.length > 0) {
      if (inComment) {
        const standardEnd = remainder.indexOf('-->');
        const alternateEnd = remainder.indexOf('--!>');
        const endings = [standardEnd, alternateEnd].filter((index) => index !== -1);
        if (endings.length === 0) {
          remainder = '';
          break;
        }
        const commentEnd = Math.min(...endings);
        const markerLength = commentEnd === alternateEnd ? 4 : 3;
        remainder = remainder.slice(commentEnd + markerLength);
        inComment = false;
        continue;
      }

      const commentStart = remainder.indexOf('<!--');
      if (commentStart === -1) {
        markdown += remainder;
        remainder = '';
      } else {
        markdown += remainder.slice(0, commentStart);
        remainder = remainder.slice(commentStart + 4);
        inComment = true;
      }
    }

    const fenceMatch = markdown.match(/^ {0,3}(`{3,}|~{3,})(.*)$/);
    if (fenceMatch && (fenceMatch[1][0] !== '`' || !fenceMatch[2].includes('`'))) {
      fence = { char: fenceMatch[1][0], length: fenceMatch[1].length };
      visible.push(null);
      continue;
    }

    visible.push(markdown.trim().length > 0 ? markdown : null);
  }

  return { lines, visible };
}

function levelTwoSections(content) {
  const { lines, visible } = structuralLines(content);
  const sections = [];
  for (let index = 0; index < lines.length; index += 1) {
    if (visible[index] === null) continue;
    const match = visible[index].match(/^ {0,3}##\s+(.+?)\s*$/);
    if (match) sections.push({ title: match[1], index });
  }
  return { lines, sections };
}

function discoverSpecFiles(root) {
  const results = [];
  if (!fs.existsSync(root)) return results;

  const walk = (directory, segments) => {
    for (const entry of fs.readdirSync(directory, { withFileTypes: true })) {
      if (entry.name.startsWith('.') || entry.isSymbolicLink()) continue;
      const entryPath = path.join(directory, entry.name);
      if (entry.isDirectory()) walk(entryPath, [...segments, entry.name]);
      else if (entry.isFile() && entry.name === 'spec.md' && segments.length > 0) {
        results.push({ capability: segments.join('/'), file: entryPath });
      }
    }
  };

  walk(root, []);
  return results.sort((left, right) => left.capability.localeCompare(right.capability));
}

function checkDeltaSections(delta) {
  const content = fs.readFileSync(delta.file, 'utf8');
  const { sections } = levelTwoSections(content);
  for (const section of sections) {
    if (!ALLOWED_DELTA_SECTIONS.some((allowed) => allowed.test(section.title))) {
      abort(`${path.relative(process.cwd(), delta.file)} contains unsupported section ` +
        `"## ${section.title}". OpenSpec 1.10.0 would ignore it during archive.`);
    }
  }
  return content;
}

function requirementScenarios(content) {
  const { lines, visible } = structuralLines(content);
  const requirements = new Map();
  let current = null;

  for (let index = 0; index < lines.length; index += 1) {
    if (visible[index] === null) continue;
    const requirement = visible[index].match(/^ {0,3}### Requirement:\s*(\S.*?)\s*$/);
    if (requirement) {
      current = { heading: requirement[1], scenarios: [] };
      if (requirements.has(current.heading)) abort(`Duplicate Requirement heading: ${current.heading}`);
      requirements.set(current.heading, current);
      continue;
    }
    if (/^ {0,3}##(?:#)?\s+/.test(visible[index]) && !/^ {0,3}#### Scenario:/.test(visible[index])) {
      if (/^ {0,3}##\s+/.test(visible[index])) current = null;
      continue;
    }
    if (!current) continue;
    const scenario = visible[index].match(/^ {0,3}#### Scenario:\s*(.*?)\s*$/);
    if (scenario) current.scenarios.push({ heading: scenario[1], start: index });
  }

  const starts = [...requirements.values()].flatMap((requirement) => requirement.scenarios)
    .sort((left, right) => left.start - right.start);
  for (let index = 0; index < starts.length; index += 1) {
    const scenario = starts[index];
    const nextScenario = starts[index + 1]?.start ?? lines.length;
    let end = nextScenario;
    for (let cursor = scenario.start + 1; cursor < nextScenario; cursor += 1) {
      if (visible[cursor]?.match(/^ {0,3}### Requirement:/) || visible[cursor]?.match(/^ {0,3}##\s+/)) {
        end = cursor;
        break;
      }
    }
    scenario.end = end;
  }
  return { lines, requirements };
}

function deltaRenames(content) {
  const renames = new Map();
  const lines = content.replace(/\r\n?/g, '\n').split('\n');
  let from = null;
  for (const line of lines) {
    const fromMatch = line.match(/^\s*-?\s*FROM:\s*`?###\s*Requirement:\s*(.+?)`?\s*$/);
    const toMatch = line.match(/^\s*-?\s*TO:\s*`?###\s*Requirement:\s*(.+?)`?\s*$/);
    if (fromMatch) from = fromMatch[1].trim();
    if (toMatch && from) {
      renames.set(toMatch[1].trim(), from);
      from = null;
    }
  }
  return renames;
}

function loadScenarioRetirements(changeDir, deltas, mainSpecsRoot) {
  const manifestFile = path.join(changeDir, 'scenario-retirements.json');
  if (!fs.existsSync(manifestFile)) return [];
  let manifest;
  try {
    manifest = JSON.parse(fs.readFileSync(manifestFile, 'utf8'));
  } catch (error) {
    abort(`Invalid scenario-retirements.json: ${error.message}`);
  }
  if (manifest.version !== 1) abort(`Unsupported scenario retirement manifest version: ${manifest.version}`);
  if (!Array.isArray(manifest.retirements)) abort('scenario-retirements.json must contain a retirements array.');

  const deltaByCapability = new Map(deltas.map((delta) => [delta.capability, delta]));
  const seen = new Set();
  const counts = new Map();
  const publishedScenarioCounts = new Map();
  const retirements = [];
  for (const entry of manifest.retirements) {
    if (!entry || typeof entry.capability !== 'string' || typeof entry.requirement !== 'string' ||
        typeof entry.scenario !== 'string') {
      abort('Every scenario retirement must contain string capability, requirement, and scenario fields.');
    }
    const identity = `${entry.capability}\0${entry.requirement}\0${entry.scenario}`;
    if (seen.has(identity)) abort(`Duplicate scenario retirement: ${entry.capability}/${entry.requirement}/${entry.scenario}`);
    seen.add(identity);
    const delta = deltaByCapability.get(entry.capability);
    if (!delta) abort(`Scenario retirement capability has no delta spec: ${entry.capability}`);
    const mainFile = path.join(mainSpecsRoot, ...entry.capability.split('/'), 'spec.md');
    if (!fs.existsSync(mainFile)) abort(`Scenario retirement capability does not exist: ${entry.capability}`);
    const main = requirementScenarios(fs.readFileSync(mainFile, 'utf8'));
    const deltaStructure = requirementScenarios(delta.content);
    const currentRequirementName = deltaRenames(delta.content).get(entry.requirement) ?? entry.requirement;
    const mainRequirement = main.requirements.get(currentRequirementName);
    const deltaRequirement = deltaStructure.requirements.get(entry.requirement);
    if (!mainRequirement) abort(`Scenario retirement Requirement does not exist: ${entry.capability}/${entry.requirement}`);
    if (!deltaRequirement) abort(`Scenario retirement Requirement is not MODIFIED by its delta: ${entry.capability}/${entry.requirement}`);
    if (!mainRequirement.scenarios.some((scenario) => scenario.heading === entry.scenario)) {
      abort(`Scenario retirement does not match the current main spec: ${entry.capability}/${entry.requirement}/${entry.scenario}`);
    }
    if (!deltaRequirement.scenarios.some((scenario) => scenario.heading === entry.scenario)) {
      abort(`Scenario retirement does not match the delta spec: ${entry.capability}/${entry.requirement}/${entry.scenario}`);
    }
    const requirementIdentity = `${entry.capability}\0${entry.requirement}`;
    counts.set(requirementIdentity, (counts.get(requirementIdentity) ?? 0) + 1);
    publishedScenarioCounts.set(requirementIdentity, deltaRequirement.scenarios.length);
    retirements.push({ ...entry, currentRequirementName, mainFile });
  }
  for (const retirement of retirements) {
    const requirementIdentity = `${retirement.capability}\0${retirement.requirement}`;
    if (publishedScenarioCounts.get(requirementIdentity) <= counts.get(requirementIdentity)) {
      abort(`Scenario retirement would leave no Scenario: ${retirement.capability}/${retirement.requirement}`);
    }
  }
  return retirements;
}

function applyScenarioRetirements(retirements) {
  const byFile = new Map();
  for (const retirement of retirements) {
    const entries = byFile.get(retirement.mainFile) ?? [];
    entries.push(retirement);
    byFile.set(retirement.mainFile, entries);
  }
  for (const [file, entries] of byFile) {
    const structure = requirementScenarios(fs.readFileSync(file, 'utf8'));
    const ranges = entries.map((entry) => {
      const requirement = structure.requirements.get(entry.requirement);
      const scenario = requirement?.scenarios.find((candidate) => candidate.heading === entry.scenario);
      if (!scenario) abort(`Published Scenario is missing during retirement: ${entry.capability}/${entry.requirement}/${entry.scenario}`);
      return scenario;
    }).sort((left, right) => right.start - left.start);
    for (const range of ranges) structure.lines.splice(range.start, range.end - range.start);
    writeAtomically(file, `${structure.lines.join('\n').trimEnd()}\n`);
  }
}

function runStructureValidation(root) {
  const command = path.join(root, 'tools', 'validate-spec-structure.mjs');
  const result = spawnSync(process.execPath, [command, root], { cwd: root, encoding: 'utf8' });
  if (result.error) abort(`Failed to execute specification structure validator: ${result.error.message}`);
  if (result.status !== 0) {
    const detail = [result.stdout, result.stderr].filter(Boolean).join('\n').trim();
    abort(`Specification structure validation failed${detail ? `:\n${detail}` : '.'}`);
  }
}

function stripHtmlComments(text) {
  const withoutClosedComments = text.replace(/<!--[\s\S]*?--!?>/g, '');
  const unterminatedComment = withoutClosedComments.indexOf('<!--');
  return unterminatedComment === -1
    ? withoutClosedComments
    : withoutClosedComments.slice(0, unterminatedComment);
}

function parseModelReplacements(modelFile) {
  const content = fs.readFileSync(modelFile, 'utf8');
  const { lines, sections } = levelTwoSections(content);
  const matches = sections.filter((section) => unnumberedTitle(section.title) === MODEL_SECTION);
  if (matches.length !== 1) abort(`model.md must contain exactly one "## ${MODEL_SECTION}" section.`);

  const start = matches[0].index + 1;
  const next = sections.find((section) => section.index >= start);
  const end = next?.index ?? lines.length;
  const sectionText = lines.slice(start, end).join('\n');
  const replacements = new Map();
  let cursor = 0;
  let noneDeclared = false;
  const sectionLines = stripHtmlComments(sectionText).split('\n');

  while (cursor < sectionLines.length) {
    const line = sectionLines[cursor];
    if (!line.trim() || /^-{3,}$/.test(line.trim())) {
      cursor += 1;
      continue;
    }
    if (line.trim() === 'None.') {
      noneDeclared = true;
      cursor += 1;
      continue;
    }
    const heading = line.match(/^###\s+`([^`]+)`\s*$/);
    if (!heading) {
      abort(`Unexpected content in "## ${MODEL_SECTION}": ${line.trim()}`);
    }

    const capability = heading[1];
    if (!CAPABILITY_PATH.test(capability)) abort(`Invalid capability path in model.md: ${capability}`);
    if (replacements.has(capability)) abort(`Duplicate Conceptual Model replacement for ${capability}.`);

    cursor += 1;
    while (cursor < sectionLines.length && !sectionLines[cursor].trim()) cursor += 1;
    if (sectionLines[cursor]?.trim() === 'REMOVE') {
      replacements.set(capability, { kind: 'remove' });
      cursor += 1;
      continue;
    }
    const opening = sectionLines[cursor]?.match(/^(`{3,}|~{3,})markdown\s*$/i);
    if (!opening) abort(`Conceptual Model replacement for ${capability} must use a markdown fenced block.`);
    const marker = opening[1];
    cursor += 1;
    const body = [];
    while (cursor < sectionLines.length && !new RegExp(`^${marker[0]}{${marker.length},}\\s*$`).test(sectionLines[cursor])) {
      body.push(sectionLines[cursor]);
      cursor += 1;
    }
    if (cursor >= sectionLines.length) abort(`Conceptual Model replacement for ${capability} has no closing fence.`);
    const replacement = body.join('\n').trim();
    if (!replacement) abort(`Conceptual Model replacement for ${capability} is empty.`);
    if (/^##\s+/m.test(replacement)) {
      abort(`Conceptual Model replacement for ${capability} must contain the section body only, without a level-two heading.`);
    }
    replacements.set(capability, { kind: 'replace', body: replacement });
    cursor += 1;
  }

  if (replacements.size === 0 && !noneDeclared) {
    abort(`"## ${MODEL_SECTION}" must contain "None." or one complete replacement per capability.`);
  }
  if (replacements.size > 0 && noneDeclared) {
    abort(`"## ${MODEL_SECTION}" cannot contain both "None." and replacements.`);
  }
  return { replacements, content };
}

function requireNoUnresolvedDecisions(modelContent) {
  const { lines, sections } = levelTwoSections(modelContent);
  const matches = sections.filter((section) => unnumberedTitle(section.title) === 'Unresolved Decisions');
  if (matches.length !== 1) abort('model.md must contain exactly one Unresolved Decisions section.');
  const start = matches[0].index + 1;
  const next = sections.find((section) => section.index >= start);
  const visible = stripHtmlComments(lines.slice(start, next?.index ?? lines.length).join('\n'))
    .split('\n')
    .filter((line) => !/^\s*-{3,}\s*$/.test(line))
    .join('\n')
    .trim();
  if (visible !== 'None.') abort('Unresolved Decisions must be exactly "None." before archive.');
}

function extractPurpose(deltaContent) {
  const { lines, sections } = levelTwoSections(deltaContent);
  const purpose = sections.find((section) => /^Purpose$/i.test(section.title));
  if (!purpose) return null;
  const next = sections.find((section) => section.index > purpose.index);
  return stripHtmlComments(lines.slice(purpose.index + 1, next?.index ?? lines.length).join('\n')).trim() || null;
}

function replaceConceptualModel(content, replacement, capability) {
  const { lines, sections } = levelTwoSections(content);
  const models = sections.filter((section) => /^Conceptual Model$/i.test(section.title));
  const requirements = sections.filter((section) => /^Requirements$/i.test(section.title));
  if (models.length > 1) abort(`${capability} has more than one Conceptual Model section.`);
  if (requirements.length !== 1) abort(`${capability} must have exactly one Requirements section.`);

  if (replacement.kind === 'remove') {
    if (models.length === 0) abort(`${capability} has no Conceptual Model section to remove.`);
    const model = models[0];
    const next = sections.find((section) => section.index > model.index);
    lines.splice(model.index, (next?.index ?? lines.length) - model.index);
    return `${lines.join('\n').trimEnd()}\n`;
  }

  const modelBlock = ['## Conceptual Model', '', replacement.body, ''];
  if (models.length === 0) {
    lines.splice(requirements[0].index, 0, ...modelBlock);
  } else {
    const model = models[0];
    const next = sections.find((section) => section.index > model.index);
    lines.splice(model.index, (next?.index ?? lines.length) - model.index, ...modelBlock);
  }
  return `${lines.join('\n').trimEnd()}\n`;
}

function newSpecSkeleton(capability, purpose, replacement) {
  if (!purpose) abort(`New capability ${capability} requires a non-empty Purpose in its delta spec.`);
  if (replacement.kind === 'remove') abort(`New capability ${capability} cannot remove a Conceptual Model.`);
  return `# ${capability} Specification\n\n## Purpose\n${purpose}\n\n` +
    `## Conceptual Model\n\n${replacement.body}\n\n## Requirements\n`;
}

function writeAtomically(file, content) {
  fs.mkdirSync(path.dirname(file), { recursive: true });
  const temporary = `${file}.conceptual-model-${process.pid}`;
  fs.writeFileSync(temporary, content, 'utf8');
  fs.renameSync(temporary, file);
}

function pruneEmptyDirectories(start, stop) {
  let current = path.dirname(start);
  const boundary = path.resolve(stop);
  while (current.startsWith(`${boundary}${path.sep}`)) {
    if (fs.readdirSync(current).length > 0) break;
    fs.rmdirSync(current);
    current = path.dirname(current);
  }
}

function captureStates(snapshots) {
  return snapshots.map(({ file }) => {
    const existed = fs.existsSync(file);
    return { file, existed, content: existed ? fs.readFileSync(file, 'utf8') : null };
  });
}

function assertStatesCurrent(states, phase) {
  for (const state of states) {
    const existed = fs.existsSync(state.file);
    const content = existed ? fs.readFileSync(state.file, 'utf8') : null;
    if (existed !== state.existed || content !== state.content) {
      abort(`Cannot ${phase}: ${state.file} changed concurrently. No files were restored.`);
    }
  }
}

function restoreSnapshots(snapshots, specsRoot, expectedStates) {
  assertStatesCurrent(expectedStates, 'restore publication state');
  for (const snapshot of [...snapshots].reverse()) {
    if (snapshot.existed) writeAtomically(snapshot.file, snapshot.content);
    else if (fs.existsSync(snapshot.file)) {
      fs.unlinkSync(snapshot.file);
      pruneEmptyDirectories(snapshot.file, specsRoot);
    }
  }
}

function assertSnapshotsCurrent(snapshots) {
  assertStatesCurrent(snapshots, 'publish validated snapshots');
}

function acquirePublicationLock(root, changeName) {
  const lockDirectory = path.join(root, 'openspec', '.archive-change.lock');
  const ownerFile = path.join(lockDirectory, 'owner.json');
  const owner = `${JSON.stringify({ pid: process.pid, change: changeName, startedAt: new Date().toISOString() })}\n`;
  try {
    fs.mkdirSync(lockDirectory);
  } catch (error) {
    if (error?.code === 'EEXIST') {
      abort(`Another publication is active (${lockDirectory}). Remove the lock only after confirming no archive process is running.`);
    }
    throw error;
  }

  try {
    fs.writeFileSync(ownerFile, owner);
  } catch (error) {
    fs.rmdirSync(lockDirectory);
    throw error;
  }
  return () => {
    if (!fs.existsSync(ownerFile) || fs.readFileSync(ownerFile, 'utf8') !== owner) {
      abort(`Publication lock ownership changed unexpectedly: ${lockDirectory}`);
    }
    fs.unlinkSync(ownerFile);
    fs.rmdirSync(lockDirectory);
  };
}

function main() {
  const [changeName, ...extra] = process.argv.slice(2);
  if (!changeName || extra.length > 0) abort('Usage: node tools/archive-change.mjs <change-name>');
  if (!CHANGE_NAME.test(changeName)) abort(`Invalid change name: ${changeName}`);

  const root = findProjectRoot(process.cwd());
  const version = runOpenSpec(root, ['--version']);
  if (version !== SUPPORTED_OPENSPEC_VERSION) {
    abort(`This publication command is verified for OpenSpec ${SUPPORTED_OPENSPEC_VERSION}; found ${version}.`);
  }

  const releaseLock = acquirePublicationLock(root, changeName);
  try {
  const changeDir = path.join(root, 'openspec', 'changes', changeName);
  const modelFile = path.join(changeDir, 'model.md');
  const changeSpecsRoot = path.join(changeDir, 'specs');
  const mainSpecsRoot = path.join(root, 'openspec', 'specs');
  if (!fs.existsSync(changeDir)) abort(`Change not found: ${changeName}`);
  if (!fs.existsSync(modelFile)) abort(`Missing model.md for change ${changeName}.`);

  const deltas = discoverSpecFiles(changeSpecsRoot);
  const deltaByCapability = new Map();
  for (const delta of deltas) {
    if (!CAPABILITY_PATH.test(delta.capability)) abort(`Invalid capability path: ${delta.capability}`);
    delta.content = checkDeltaSections(delta);
    deltaByCapability.set(delta.capability, delta);
  }

  const parsedModel = parseModelReplacements(modelFile);
  const { replacements } = parsedModel;
  requireNoUnresolvedDecisions(parsedModel.content);
  for (const capability of replacements.keys()) {
    if (!deltaByCapability.has(capability)) {
      abort(`Conceptual Model replacement for ${capability} requires a delta spec at ` +
        `openspec/changes/${changeName}/specs/${capability}/spec.md.`);
    }
  }
  const retirements = loadScenarioRetirements(changeDir, deltas, mainSpecsRoot);

  runOpenSpec(root, ['schema', 'validate', 'quality-spec']);
  runOpenSpec(root, ['validate', '--specs', '--strict', '--json']);
  runOpenSpec(root, ['validate', changeName, '--strict', '--json']);

  const status = JSON.parse(runOpenSpec(root, ['status', '--change', changeName, '--json']));
  if (!status.isPlanningComplete) abort(`Change ${changeName} has incomplete planning artifacts.`);
  const apply = JSON.parse(runOpenSpec(root, ['instructions', 'apply', '--change', changeName, '--json']));
  if (!apply.progress || apply.progress.total === 0) {
    abort(`Change ${changeName} has no implementation or verification tasks.`);
  }
  if (apply.progress?.remaining !== 0) {
    abort(`Change ${changeName} has ${apply.progress?.remaining ?? 'unknown'} incomplete task(s).`);
  }

  const snapshots = deltas.map((delta) => {
    const file = path.join(mainSpecsRoot, ...delta.capability.split('/'), 'spec.md');
    const existed = fs.existsSync(file);
    return { file, existed, content: existed ? fs.readFileSync(file, 'utf8') : null };
  });

  for (const delta of deltas) {
    const target = path.join(mainSpecsRoot, ...delta.capability.split('/'), 'spec.md');
    const purpose = extractPurpose(delta.content);
    if (fs.existsSync(target) && purpose) {
      abort(`Existing capability ${delta.capability} must omit Purpose from its delta spec; OpenSpec would ignore it.`);
    }
    if (!fs.existsSync(target) && !purpose) {
      abort(`New capability ${delta.capability} requires Purpose in its delta spec.`);
    }
  }

  const stagedSnapshots = [];
  const stagedStates = [];
  let archiveResult = null;
  let publishedStates = null;
  try {
    for (const [capability, replacement] of replacements.entries()) {
      const delta = deltaByCapability.get(capability);
      const target = path.join(mainSpecsRoot, ...capability.split('/'), 'spec.md');
      const snapshot = snapshots.find(({ file }) => file === target);
      assertSnapshotsCurrent([snapshot]);
      const purpose = extractPurpose(delta.content);
      const staged = snapshot.existed
        ? replaceConceptualModel(snapshot.content, replacement, capability)
        : newSpecSkeleton(capability, purpose, replacement);
      writeAtomically(target, staged);
      stagedSnapshots.push(snapshot);
      stagedStates.push({ file: target, existed: true, content: staged });
    }
    assertStatesCurrent(stagedStates, 'start OpenSpec archive');

    const archiveOutput = runOpenSpec(root, ['archive', changeName, '--yes', '--json']);
    archiveResult = JSON.parse(archiveOutput);
    if (!archiveResult.archive?.path) abort('OpenSpec did not return an archive path.');
    applyScenarioRetirements(retirements);
    publishedStates = captureStates(snapshots);
    runOpenSpec(root, ['validate', '--specs', '--strict', '--json']);
    runStructureValidation(root);

    process.stdout.write(`${JSON.stringify(archiveResult, null, 2)}\n`);
  } catch (error) {
    if (archiveResult?.archive?.path && !fs.existsSync(changeDir)) {
      restoreSnapshots(snapshots, mainSpecsRoot, publishedStates);
      fs.renameSync(archiveResult.archive.path, changeDir);
    } else if (fs.existsSync(changeDir) && stagedSnapshots.length > 0) {
      restoreSnapshots(stagedSnapshots, mainSpecsRoot, stagedStates);
    }
    throw error;
  }
  } finally {
    releaseLock();
  }
}

try {
  main();
} catch (error) {
  process.stderr.write(`archive-change: ${error instanceof Error ? error.message : String(error)}\n`);
  process.exitCode = 1;
}
