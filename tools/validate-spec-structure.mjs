#!/usr/bin/env node

import fs from 'node:fs';
import path from 'node:path';
import process from 'node:process';

function fail(message) {
  throw new Error(message);
}

function visibleLines(content) {
  const lines = content.replace(/\r\n?/g, '\n').split('\n');
  const visible = [];
  let fence = null;
  let inComment = false;

  for (const line of lines) {
    if (fence) {
      if (new RegExp(`^ {0,3}${fence.char}{${fence.length},}\\s*$`).test(line)) fence = null;
      visible.push(null);
      continue;
    }

    let remainder = line;
    let markdown = '';
    while (remainder.length > 0) {
      if (inComment) {
        const ends = [remainder.indexOf('-->'), remainder.indexOf('--!>')].filter((index) => index >= 0);
        if (ends.length === 0) {
          remainder = '';
          break;
        }
        const end = Math.min(...ends);
        remainder = remainder.slice(end + (end === remainder.indexOf('--!>') ? 4 : 3));
        inComment = false;
        continue;
      }
      const start = remainder.indexOf('<!--');
      if (start < 0) {
        markdown += remainder;
        remainder = '';
      } else {
        markdown += remainder.slice(0, start);
        remainder = remainder.slice(start + 4);
        inComment = true;
      }
    }

    const opening = markdown.match(/^ {0,3}(`{3,}|~{3,})(.*)$/);
    if (opening && (opening[1][0] !== '`' || !opening[2].includes('`'))) {
      fence = { char: opening[1][0], length: opening[1].length };
      visible.push(null);
      continue;
    }
    visible.push(markdown.trim() ? markdown : null);
  }
  return visible;
}

function discoverSpecs(specsRoot) {
  const files = [];
  const walk = (directory) => {
    for (const entry of fs.readdirSync(directory, { withFileTypes: true })) {
      if (entry.name.startsWith('.') || entry.name === '_schema' || entry.isSymbolicLink()) continue;
      const target = path.join(directory, entry.name);
      if (entry.isDirectory()) walk(target);
      else if (entry.isFile() && entry.name === 'spec.md') files.push(target);
    }
  };
  if (fs.existsSync(specsRoot)) walk(specsRoot);
  return files.sort();
}

function validateFile(root, file, structure) {
  const relative = path.relative(root, file);
  const lines = visibleLines(fs.readFileSync(file, 'utf8'));
  const blockIndexes = new Map(structure.blocks.map((block, index) => [block.label, index]));
  const allowedTags = new Set(structure.scenario.allowed_tags);
  const slugPattern = new RegExp(structure.requirement_slug_pattern);
  let requirement = null;

  const problem = (index, message) => fail(`${relative}:${index + 1}: ${message}`);

  for (let index = 0; index < lines.length; index += 1) {
    const line = lines[index];
    if (line === null) continue;

    const requirementMatch = line.match(/^ {0,3}### Requirement:\s*(\S.*?)\s*$/);
    if (requirementMatch) {
      const slug = requirementMatch[1];
      if (!slugPattern.test(slug)) problem(index, `invalid Requirement slug "${slug}".`);
      requirement = { slug, lastBlock: -1, blocks: new Set(), hasScenario: false };
      continue;
    }
    if (/^ {0,3}##(?:#)?\s+/.test(line) && !/^ {0,3}#### Scenario:/.test(line)) {
      if (/^ {0,3}##\s+/.test(line)) requirement = null;
      continue;
    }
    if (!requirement) continue;

    const scenarioMatch = line.match(/^ {0,3}#### Scenario:\s*(.*?)\s*$/);
    if (scenarioMatch) {
      requirement.hasScenario = true;
      const tag = scenarioMatch[1].match(/\[([^\]]+)\]$/)?.[1];
      if (!tag) problem(index, 'Scenario heading must end with one allowed tag.');
      if (!allowedTags.has(tag)) problem(index, `unknown Scenario tag "${tag}".`);
      continue;
    }

    const blockMatch = line.match(/^\s*-\s+\*\*([^*]+)\*\*:/);
    if (!blockMatch || ['GIVEN', 'WHEN', 'THEN'].includes(blockMatch[1])) continue;
    const label = blockMatch[1];
    if (!blockIndexes.has(label)) problem(index, `unknown Requirement block "${label}".`);
    if (requirement.hasScenario) problem(index, `Requirement block "${label}" must precede Scenarios.`);
    if (requirement.blocks.has(label)) problem(index, `Requirement repeats Requirement block "${label}".`);
    const blockIndex = blockIndexes.get(label);
    if (blockIndex < requirement.lastBlock) problem(index, `Requirement block "${label}" is out of order.`);
    requirement.blocks.add(label);
    requirement.lastBlock = blockIndex;
  }
}

function main() {
  const [rootArgument, ...extra] = process.argv.slice(2);
  if (extra.length > 0) fail('Usage: node tools/validate-spec-structure.mjs [repository-root]');
  const root = path.resolve(rootArgument ?? process.cwd());
  const specsRoot = path.join(root, 'openspec', 'specs');
  const structureFile = path.join(specsRoot, '_schema', 'requirement-structure.json');
  if (!fs.existsSync(structureFile)) fail(`Missing requirement structure: ${structureFile}`);
  const structure = JSON.parse(fs.readFileSync(structureFile, 'utf8'));
  if (!Array.isArray(structure.blocks) || !Array.isArray(structure.scenario?.allowed_tags)) {
    fail(`Invalid requirement structure: ${structureFile}`);
  }
  const files = discoverSpecs(specsRoot);
  for (const file of files) validateFile(root, file, structure);
  process.stdout.write(`Validated ${files.length} main spec${files.length === 1 ? '' : 's'}.\n`);
}

try {
  main();
} catch (error) {
  process.stderr.write(`validate-spec-structure: ${error instanceof Error ? error.message : String(error)}\n`);
  process.exitCode = 1;
}
