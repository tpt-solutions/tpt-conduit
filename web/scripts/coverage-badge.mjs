// Generates a shields.io-style SVG coverage badge from vitest's json-summary
// coverage report. Run after `npm run test:coverage` (reportsDirectory: coverage).
import { readFileSync, writeFileSync, mkdirSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { dirname, resolve } from "node:path";

const root = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const summaryPath = resolve(root, "coverage", "coverage-summary.json");
const outPath = resolve(root, "coverage", "badge.svg");

function colorFor(pct) {
  if (pct >= 80) return "#4caf50";
  if (pct >= 50) return "#dfb317";
  return "#e05d44";
}

function escapeXml(s) {
  return String(s).replace(/[<>&"]/g, (c) =>
    ({ "<": "&lt;", ">": "&gt;", "&": "&amp;", '"': "&quot;" }[c])
  );
}

function badge(label, value, color) {
  const labelText = escapeXml(label);
  const valueText = escapeXml(value);
  const labelWidth = labelText.length * 7 + 10;
  const valueWidth = valueText.length * 7 + 10;
  const total = labelWidth + valueWidth;
  return `<svg xmlns="http://www.w3.org/2000/svg" width="${total}" height="20" role="img" aria-label="${labelText}: ${valueText}">
  <linearGradient id="s" x2="0" y2="100%">
    <stop offset="0" stop-color="#bbb" stop-opacity=".1"/>
    <stop offset="1" stop-opacity=".1"/>
  </linearGradient>
  <clipPath id="r"><rect width="${total}" height="20" rx="3" fill="#fff"/></clipPath>
  <g clip-path="url(#r)">
    <rect width="${labelWidth}" height="20" fill="#555"/>
    <rect x="${labelWidth}" width="${valueWidth}" height="20" fill="${color}"/>
    <rect width="${total}" height="20" fill="url(#s)"/>
  </g>
  <g fill="#fff" text-anchor="middle" font-family="Verdana,Geneva,DejaVu Sans,sans-serif" font-size="11">
    <text x="${labelWidth / 2}" y="14">${labelText}</text>
    <text x="${labelWidth + valueWidth / 2}" y="14">${valueText}</text>
  </g>
</svg>`;
}

try {
  const summary = JSON.parse(readFileSync(summaryPath, "utf8"));
  const pct = summary.total.statements.pct;
  const value = Number.isFinite(pct) ? `${pct.toFixed(0)}%` : "unknown";
  mkdirSync(dirname(outPath), { recursive: true });
  writeFileSync(outPath, badge("coverage", value, colorFor(pct)), "utf8");
  console.log(`Wrote coverage badge: ${value}`);
} catch (err) {
  console.error(`Could not generate coverage badge: ${err.message}`);
  process.exit(1);
}
