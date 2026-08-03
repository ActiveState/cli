const os = require('os');
const path = require('path');
const process = require('process');

// WARNING! THIS FILE MUST BE KEPT IN SYNC WITH `internal/installation/paths_*.go`

const installPath = () => {
    switch (process.platform) {
        case 'win32':
            return path.join(process.env.USERPROFILE, "AppData", "Local", "ActiveState", "StateTool", "release");
        case 'darwin':
            return path.join(os.homedir(), ".local", "ActiveState", "StateTool", "release");
        case 'linux':
            return path.join(os.homedir(), ".local", "ActiveState", "StateTool", "release");
        default:
            throw new Error(`Unsupported platform: ${process.platform}`);
    }
};

const binPath = () => {
    let ext = "";
    if (process.platform === 'win32') {
        ext = ".exe";
    }
    return path.join(installPath(), "bin", "state" + ext);
};

module.exports = {
    installPath: installPath,
    binPath: binPath,
};