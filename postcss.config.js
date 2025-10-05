import { purgeCSSPlugin } from "@fullhuman/postcss-purgecss";
import postcssPresetEnv from "postcss-preset-env";
import postcssImport from "postcss-import";
import cssnano from "cssnano";

export default {
    plugins: [
        postcssImport,
        purgeCSSPlugin({
            content: ["./src/**/*.hbs", "./src/**/*.js"],
        }),
        postcssPresetEnv({
            browsers: "last 4 versions",
            // false *dsables* polyfills
            features: {
                "cascade-layers": false,
            },
            autoprefixer: false,
        }),
        cssnano,
    ],
};
