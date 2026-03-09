// ─── HELM Docs Proof Linter ──────────────────────────────────────────────────
// Ensures doc pages tagged with "core claims" contain at least one proof link.
//
// A "proof link" is any of:
//   - Link to a demo route (e.g., /demos/, use-cases/)
//   - Link to a release asset (e.g., GitHub release URL)
//   - Command snippet that produces verifiable output
//   - Link to an Evidence Pack fixture
//
// Usage: npx tsx qa/tools/docs-proof-lint.ts [docs_dir]

import { readFileSync, readdirSync, statSync } from "node:fs";
import { join, relative, extname } from "node:path";

// ─── Configuration ───────────────────────────────────────────────────────────

/** Pages that MUST contain at least one proof link. */
const PROOF_REQUIRED_PATTERNS = [
    /^README\.md$/i,
    /START_HERE/i,
    /docs\/use-cases\//i,
    /docs\/cli/i,
    /VERIFY/i,
];

/** Patterns that count as "proof links". */
const PROOF_LINK_PATTERNS = [
    /\bdemos?\//i,                            // demo route
    /\buse-cases?\//i,                        // use case link
    /github\.com\/.*\/releases/i,             // GitHub release URL
    /npx\s+@mindburn\/helm/i,                // CLI command snippet
    /fixtures\//i,                            // Evidence Pack fixture link
    /```(?:bash|sh|shell)[\s\S]*?helm/i,      // shell snippet mentioning helm
    /evidence[_-]?pack/i,                     // Evidence Pack mention
    /EXPECTED\.json/i,                        // fixture reference
];

// ─── Types ───────────────────────────────────────────────────────────────────

interface LintResult {
    path: string;
    requiresProof: boolean;
    hasProofLink: boolean;
    proofLinksFound: string[];
}

interface LintReport {
    total_files: number;
    files_requiring_proof: number;
    files_with_proof: number;
    files_missing_proof: number;
    pass: boolean;
    results: LintResult[];
}

// ─── Scanner ─────────────────────────────────────────────────────────────────

function scanMarkdownFiles(dir: string): string[] {
    const results: string[] = [];
    const entries = readdirSync(dir, { withFileTypes: true });
    for (const entry of entries) {
        const fullPath = join(dir, entry.name);
        if (entry.isDirectory() && !entry.name.startsWith(".") && entry.name !== "node_modules") {
            results.push(...scanMarkdownFiles(fullPath));
        } else if (entry.isFile() && extname(entry.name) === ".md") {
            results.push(fullPath);
        }
    }
    return results;
}

function requiresProof(relativePath: string): boolean {
    return PROOF_REQUIRED_PATTERNS.some((p) => p.test(relativePath));
}

function findProofLinks(content: string): string[] {
    const found: string[] = [];
    for (const pattern of PROOF_LINK_PATTERNS) {
        const match = content.match(pattern);
        if (match) {
            found.push(match[0].trim().substring(0, 80));
        }
    }
    return found;
}

// ─── Main ────────────────────────────────────────────────────────────────────

function lint(projectRoot: string): LintReport {
    const docsDir = join(projectRoot, "docs");
    const readmePath = join(projectRoot, "README.md");

    const allFiles: string[] = [];
    if (statSync(readmePath, { throwIfNoEntry: false })?.isFile()) {
        allFiles.push(readmePath);
    }
    if (statSync(docsDir, { throwIfNoEntry: false })?.isDirectory()) {
        allFiles.push(...scanMarkdownFiles(docsDir));
    }

    const results: LintResult[] = [];

    for (const file of allFiles) {
        const rel = relative(projectRoot, file);
        const needs = requiresProof(rel);
        const content = readFileSync(file, "utf-8");
        const proofs = needs ? findProofLinks(content) : [];

        results.push({
            path: rel,
            requiresProof: needs,
            hasProofLink: proofs.length > 0,
            proofLinksFound: proofs,
        });
    }

    const requiring = results.filter((r) => r.requiresProof);
    const withProof = requiring.filter((r) => r.hasProofLink);
    const missing = requiring.filter((r) => !r.hasProofLink);

    return {
        total_files: results.length,
        files_requiring_proof: requiring.length,
        files_with_proof: withProof.length,
        files_missing_proof: missing.length,
        pass: missing.length === 0,
        results,
    };
}

// ─── CLI ─────────────────────────────────────────────────────────────────────

const projectRoot = process.argv[2] || process.cwd();
const report = lint(projectRoot);

// Output JSON report
const outDir = join(projectRoot, "artifacts", "site_audit");
import { mkdirSync, writeFileSync } from "node:fs";
mkdirSync(outDir, { recursive: true });
writeFileSync(join(outDir, "docs_proof_map.json"), JSON.stringify(report, null, 2) + "\n");

// Console output
const icon = report.pass ? "✅" : "❌";
console.log(`${icon} Docs Proof Lint: ${report.files_with_proof}/${report.files_requiring_proof} pages have proof links`);

if (!report.pass) {
    console.log("\nPages missing proof links:");
    for (const r of report.results.filter((r) => r.requiresProof && !r.hasProofLink)) {
        console.log(`  ❌ ${r.path}`);
    }
    process.exit(1);
}
