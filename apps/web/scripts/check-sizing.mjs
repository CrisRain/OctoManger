import fs from "node:fs";
import path from "node:path";

const sourceRoot = path.join(process.cwd(), "src");
const allowedExtensions = new Set([".vue", ".ts", ".tsx", ".scss", ".css"]);
const excludedFiles = new Set([
  path.join(sourceRoot, "auto-imports.d.ts"),
  path.join(sourceRoot, "lib", "icons.ts"),
  path.join(sourceRoot, "styles", "_tailwind-utilities.scss"),
  path.join(sourceRoot, "styles", "tailwind.css"),
]);

const sizePropertyPattern =
  /(?<![\w-])(?:width|height|min-width|min-height|max-width|max-height|margin(?:-(?:top|right|bottom|left|inline|block|inline-start|inline-end|block-start|block-end))?|padding(?:-(?:top|right|bottom|left|inline|block|inline-start|inline-end|block-start|block-end))?|gap|column-gap|row-gap|top|right|bottom|left|inset(?:-(?:top|right|bottom|left|inline|block|inline-start|inline-end|block-start|block-end))?|inline-size|block-size|min-inline-size|max-inline-size|min-block-size|max-block-size|grid-template-columns|grid-template-rows|flex-basis)\s*:\s*([^;]+)/i;

const absoluteUnitPattern = /(-?\d*\.?\d+)(px|rem)\b/gi;
const numericDimensionBindingPattern = /:\s*(width|height)\s*=\s*["']\d+(?:\.\d+)?["']/;
const numericDimensionOptionPattern = /(?<![\w-])(width|height)\s*:\s*\d+(?:\.\d+)?\s*(?:,|}|$)/;

function walk(dir) {
  const entries = fs.readdirSync(dir, { withFileTypes: true });
  const files = [];

  for (const entry of entries) {
    const fullPath = path.join(dir, entry.name);
    if (entry.isDirectory()) {
      files.push(...walk(fullPath));
      continue;
    }
    if (allowedExtensions.has(path.extname(entry.name)) && !excludedFiles.has(fullPath)) {
      files.push(fullPath);
    }
  }

  return files;
}

function hasNonZeroAbsoluteUnit(value) {
  for (const match of value.matchAll(absoluteUnitPattern)) {
    if (Number.parseFloat(match[1]) !== 0) {
      return true;
    }
  }
  return false;
}

function inspectLine(filePath, line, lineNumber) {
  const findings = [];
  const propertyMatch = line.includes("@media") ? null : line.match(sizePropertyPattern);

  if (propertyMatch && hasNonZeroAbsoluteUnit(propertyMatch[1])) {
    findings.push({
      filePath,
      lineNumber,
      message: "absolute unit used in sizing property",
      line,
    });
  }

  if (numericDimensionBindingPattern.test(line)) {
    findings.push({
      filePath,
      lineNumber,
      message: "numeric width/height binding found",
      line,
    });
  }

  if (numericDimensionOptionPattern.test(line)) {
    findings.push({
      filePath,
      lineNumber,
      message: "numeric width/height option found",
      line,
    });
  }

  return findings;
}

const findings = [];

for (const filePath of walk(sourceRoot)) {
  const content = fs.readFileSync(filePath, "utf8");
  const lines = content.split(/\r?\n/);

  lines.forEach((line, index) => {
    findings.push(...inspectLine(filePath, line, index + 1));
  });
}

if (findings.length > 0) {
  console.error("Sizing policy violations detected:");
  for (const finding of findings) {
    const relativePath = path.relative(process.cwd(), finding.filePath);
    console.error(`- ${relativePath}:${finding.lineNumber} ${finding.message}`);
    console.error(`  ${finding.line.trim()}`);
  }
  process.exit(1);
}

console.log("Sizing policy check passed.");
