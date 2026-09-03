/**
 * @fileoverview NodeJS binding to run the buildozer command through a convenient API
 */
const {spawnSync} = require('node:child_process');
// bazel will copy the /launcher.js in this repo while building the package
const {getNativeBinary} = require('./buildozer');

/**
 * run buildozer with a list of commands
 * @see https://github.com/bazel-contrib/buildtools/tree/main/buildozer#do-multiple-changes-at-once
 * @returns The standard out of the buildozer command, split by lines
 */
function run(...commands) {
    return runWithOptions(commands, {});
}

/**
 * run buildozer with a list of commands
 * @param commands a list of CommandBatch to pass to buildozer
 * @param options Options to pass to spawn
 * @param flags any buildozer flags to pass
 */
function runWithOptions(commands, options, flags = []) {
    // From https://github.com/bazel-contrib/buildtools/tree/main/buildozer#usage:
    // Here, label-list is a space-separated list of Bazel labels,
    // for example //path/to/pkg1:rule1 //path/to/pkg2:rule2.
    // Buildozer reads commands from FILE (- for stdin 
    //   (format: |-separated command line arguments to buildozer, excluding flags))
    const input = commands.map(c => [...c.commands, c.targets.join(',')].join('|')).join('\n');
    const {stdout, stderr, status, error} = spawnSync(getNativeBinary(), flags.concat([
        '-f', '-' /* read commands from stdin */
    ]), {
        ...options,
        input,
        encoding: 'utf-8',
    });
    // https://github.com/bazel-contrib/buildtools/tree/main/buildozer#error-code
    if (status == 0 || status == 3) return stdout.trim().split('\n').filter(l => !!l);

    console.error(`buildozer exited with status ${status}\n${stderr}`);
    throw error;
}

module.exports = {
    runWithOptions,
    run,
};
