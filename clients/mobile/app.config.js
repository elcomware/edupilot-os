// Expo prebuild consumes the same artwork on developer machines and CI.
const { relative, sep } = require("node:path");
const asset = (name) => relative(__dirname, require.resolve("@edupilot/core/brand/platform/" + name))
  .split(sep).join("/");

module.exports = ({ config }) => ({
  ...config,
  icon: asset("apple-icon-1024.png"),
  android: {
    ...config.android,
    adaptiveIcon: {
      foregroundImage: asset("android-adaptive-foreground-432.png"),
      backgroundImage: asset("android-adaptive-background-432.png"),
      monochromeImage: asset("android-adaptive-monochrome-432.png"),
    },
  },
});
