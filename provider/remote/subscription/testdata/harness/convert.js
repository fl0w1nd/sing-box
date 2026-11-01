// Converts every testdata/cases/*.txt subscription with Sub-Store and writes the
// sing-box outbounds next to it as *.json.
//
// Usage: node convert.js <Sub-Store backend directory> <cases directory>
// The backend directory needs its dependencies installed (pnpm install).
'use strict';

const fs = require('fs');
const path = require('path');

const backend = path.resolve(process.argv[2]);
const casesDirectory = path.resolve(process.argv[3]);

// Sub-Store resolves "@/..." imports relative to the working directory.
process.chdir(backend);
require(path.join(backend, 'node_modules/@babel/register'))({
    cwd: backend,
    root: backend,
    only: [path.join(backend, 'src')],
});

const { ProxyUtils } = require(path.join(backend, 'src/core/proxy-utils'));
const $ = require(path.join(backend, 'src/core/app')).default;
for (const level of ['info', 'log', 'warn', 'error', 'notify']) {
    $[level] = () => {};
}

// Port hopping picks a random port; the Go port keeps the first one. Set after
// loading Sub-Store, since Babel needs a working Math.random.
Math.random = () => 0;

for (const name of fs.readdirSync(casesDirectory).sort()) {
    if (!name.endsWith('.txt')) continue;
    const raw = fs.readFileSync(path.join(casesDirectory, name), 'utf8');
    let outbounds;
    try {
        const produced = JSON.parse(
            ProxyUtils.produce(ProxyUtils.parse(raw), 'sing-box'),
        );
        outbounds = produced.outbounds;
    } catch (e) {
        outbounds = { error: `${e}` };
    }
    fs.writeFileSync(
        path.join(casesDirectory, name.replace(/\.txt$/, '.json')),
        JSON.stringify({ outbounds }, null, 2) + '\n',
    );
}
