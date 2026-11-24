const fs = require('fs');
const os = require('os');
const package = require('./package.json');
const path = require('path');
const requestp = require('request-promise');
const decompress = require('decompress');
const sha256 = require('js-sha256');
const child_process = require('child_process');
const process = require('process');


module.exports = async function (callback) {
    let version = package.version.split('-RC')[0];

    let platform = {
        darwin_x64: 'darwin-amd64',
        darwin_arm64: 'darwin-amd64',
        win32_x64: 'windows-amd64',
        linux_x64: 'linux-amd64'
    }[process.platform + '_' + process.arch];

    if (!platform) {
        throw new Error('Unsupported OS: ' + process.platform + '-' + process.arch);
    }

    console.log(`Preparing Installer for State Tool Package Manager version ${version}`);

    let ext = "tar.gz";
    if (process.platform === 'win32') {
        ext = "zip";
    }

    let payloadURL = `https://state-tool.s3.amazonaws.com/update/state/release/${version}/${platform}/state-${platform}-${version}.${ext}`;
    let payload = await request({method: 'GET', url: payloadURL, encoding: null});

    // Verify Hash
    let infoURL = `https://state-tool.s3.amazonaws.com/update/state/release/${version}/${platform}/info.json`;
    let info = await request(infoURL);
    info = JSON.parse(info);
    let hash = sha256(payload);
    if (hash !== info.sha256) {
        throw new Error(`Hash mismatch: ${hash} !== ${info.sha256}`);
    }

    let tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "activestate-cli"));
    await decompress(payload, tmpDir, {strip: 1});

    let exeExt = "";
    if (process.platform === 'win32') {
        exeExt = ".exe";
    }
    let env = Object.create(process.env);
    //env.VERBOSE = true;
    let proc = child_process
        .spawnSync('"' + path.join(tmpDir, "state-installer" + exeExt) + '"', ["--source-installer", "install.js", "--source-path", tmpDir], {
            stdio: 'inherit',
            shell: true,
            env: env,
        });

    fs.rmSync(tmpDir, {recursive: true});
    process.exit(proc.status);
};


async function request() {
    let result;
    await requestp.apply(null, arguments)
        .then(res => {
            result = res;
        })
        .catch(err => {
            throw new Error(`Error downloading ${JSON.stringify(arguments)}: ${err}`);
        });
    return result;
}