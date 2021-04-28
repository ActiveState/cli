const path = require('path');

module.exports = {
    entry: ['babel-polyfill', './main.js'],
    target: "es5",
    mode: 'development',
    output: {
        filename: 'main.js',
        path: path.resolve(__dirname, 'generated')
    },
    module: {
        rules: [{
            test: /\.js$/,
            exclude: /node_modules/,
            use: {
                loader: 'babel-loader',
                options: {
                    babelrc: true
                }
            }
        }]
    },
};