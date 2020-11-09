const UglifyJsPlugin = require("uglifyjs-webpack-plugin");

module.exports = {
  entry: {
    entry: './src/index.js',
  },
  output: {
    filename: "build/static/js/bundle.js",
  },
  module: {
    rules: [
      {
        test: /\.css$/,
        use: ["style-loader", "css-loader"],
      },
      {
        test: /\.m?js$/,
        exclude: /(node_modules)/,
        use: {
          loader: 'babel-loader',
          options: {
            presets: ['@babel/preset-env'],
            plugins: ['@babel/plugin-transform-runtime']
          }
        }
      }
    ],
  },
  plugins: [new UglifyJsPlugin()],
}
