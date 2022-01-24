const path = require('path');

module.exports = {
    entry: ['babel-polyfill', './main.js'],
    target: "es5",
    mode: 'production', // IMPORTANT: When changing this during development, ensure to switch back to production in the end, as otherwise the dialog will fail to load on Windows
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
