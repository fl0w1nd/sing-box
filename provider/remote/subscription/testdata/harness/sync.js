// Checks the Go port against a Sub-Store revision.
//
// Usage:
//   node sync.js [--ref <branch, tag or commit>] [--update] [--report <file>]
//
// The script clones Sub-Store at ref (default: master), installs its backend
// dependencies with pnpm, and regenerates testdata/cases/*.json with that
// revision. It then compares the Sub-Store files the port follows with the
// pinned revision in UPSTREAM.json and writes a Markdown report.
//
// After it, `go test ./provider/remote/subscription/` shows whether the port
// still matches Sub-Store, and `git diff` shows how the output changed.
//
// With --update, UPSTREAM.json is pinned to ref. Only pin after the port has
// been updated to match: the pin records which revision the Go code follows.
//
// Requires git, Node.js and pnpm.
'use strict';

const childProcess = require('child_process');
const crypto = require('crypto');
const fs = require('fs');
const os = require('os');
const path = require('path');

const packageDirectory = path.resolve(__dirname, '..', '..');
const casesDirectory = path.join(packageDirectory, 'testdata', 'cases');
const upstreamFile = path.join(packageDirectory, 'UPSTREAM.json');

function parseArguments() {
    const options = { ref: 'master', update: false, report: '' };
    const args = process.argv.slice(2);
    for (let i = 0; i < args.length; i++) {
        switch (args[i]) {
            case '--ref':
                options.ref = args[++i];
                break;
            case '--update':
                options.update = true;
                break;
            case '--report':
                options.report = args[++i];
                break;
            default:
                throw new Error(`unknown argument: ${args[i]}`);
        }
    }
    return options;
}

function run(command, args, cwd, options = {}) {
    return childProcess.execFileSync(command, args, {
        cwd,
        encoding: 'utf8',
        stdio: options.inherit ? 'inherit' : ['ignore', 'pipe', 'inherit'],
        maxBuffer: 64 << 20,
    });
}

function sha256(file) {
    return crypto.createHash('sha256').update(fs.readFileSync(file)).digest('hex');
}

function main() {
    const options = parseArguments();
    const upstream = JSON.parse(fs.readFileSync(upstreamFile, 'utf8'));
    const checkout = fs.mkdtempSync(path.join(os.tmpdir(), 'sub-store-'));
    try {
        run('git', ['init', '-q'], checkout);
        run('git', ['remote', 'add', 'origin', upstream.repository], checkout);
        run('git', ['fetch', '-q', '--depth=1', 'origin', options.ref], checkout, { inherit: true });
        run('git', ['checkout', '-q', 'FETCH_HEAD'], checkout);
        const commit = run('git', ['rev-parse', 'HEAD'], checkout).trim();
        const backend = path.join(checkout, 'backend');
        run('pnpm', ['install', '--frozen-lockfile', '--ignore-scripts', '--silent'], backend, { inherit: true });
        run('node', [path.join(__dirname, 'convert.js'), backend, casesDirectory], backend, { inherit: true });

        const files = {};
        const changedFiles = [];
        for (const file of Object.keys(upstream.files)) {
            const absolute = path.join(checkout, file);
            files[file] = fs.existsSync(absolute) ? sha256(absolute) : null;
            if (files[file] !== upstream.files[file]) {
                changedFiles.push(file);
            }
        }

        let diff = '';
        if (changedFiles.length > 0 && commit !== upstream.commit) {
            try {
                run('git', ['fetch', '-q', '--depth=1', 'origin', upstream.commit], checkout);
                diff = run('git', ['diff', upstream.commit, commit, '--', ...changedFiles], checkout);
            } catch (e) {
                diff = `(cannot diff against ${upstream.commit}: ${e.message})`;
            }
        }

        const lines = [
            `Sub-Store ${upstream.repository}/commit/${commit}`,
            `Pinned: ${upstream.repository}/commit/${upstream.commit}`,
            '',
        ];
        if (changedFiles.length === 0) {
            lines.push('The files the port follows are unchanged.');
        } else {
            lines.push('Files the port follows that changed since the pinned commit:', '');
            for (const file of changedFiles) {
                lines.push(`- \`${file}\``);
            }
            if (diff) {
                lines.push('', '<details><summary>Diff</summary>', '', '```diff', diff.slice(0, 60000), '```', '', '</details>');
            }
        }
        const report = lines.join('\n') + '\n';
        if (options.report) {
            fs.writeFileSync(options.report, report);
        }
        process.stdout.write(report);

        if (options.update) {
            upstream.commit = commit;
            upstream.files = files;
            fs.writeFileSync(upstreamFile, JSON.stringify(upstream, null, 2) + '\n');
        }
        if (changedFiles.length > 0 && !options.update) {
            process.exitCode = 2;
        }
    } finally {
        fs.rmSync(checkout, { recursive: true, force: true });
    }
}

main();
