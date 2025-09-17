const process = require('process');
const fs = require('fs');
const paths = require("../paths.js");

if (!process.env.npm_config_global) {
    console.error('Must be installed globally (use -g flag).');
    process.exit(1);
}

const binaryPath = paths.binPath();

if (fs.existsSync(binaryPath)) {
    console.error(`State Tool is already installed at: ${binaryPath}.`);
    process.exit(1);
}

require('../install.js')();