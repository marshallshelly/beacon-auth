// @ts-check
import { defineConfig } from "astro/config";
import starlight from "@astrojs/starlight";
import path from "node:path";
import { fileURLToPath } from "node:url";

const __dirname = path.dirname(fileURLToPath(import.meta.url));

// https://astro.build/config
export default defineConfig({
  site: "https://marshallshelly.github.io/beacon-auth",
  base: "/beacon-auth/",
  vite: {
    resolve: {
      alias: {
        "@astrojs/starlight/components": path.resolve(
          __dirname,
          "./node_modules/@astrojs/starlight/components"
        ),
      },
    },
  },
  integrations: [
    starlight({
      title: "BeaconAuth",
      description:
        "A modular, plugin-based authentication library for Go. Secure, flexible, and built for modern apps.",
      social: [
        {
          icon: "github",
          label: "GitHub",
          href: "https://github.com/marshallshelly/beacon-auth",
        },
      ],
      editLink: {
        baseUrl: "https://github.com/marshallshelly/beacon-auth/",
      },
      customCss: ["./src/styles/custom.css"],
      sidebar: [
        {
          label: "Start Here",
          items: [
            {
              label: "Quickstart",
              slug: "getting-started/quickstart",
            },
          ],
        },
        {
          label: "Guides",
          items: [
            {
              label: "Plugins System",
              slug: "guides/plugins",
            },
          ],
        },
        {
          label: "Reference",
          items: [
            {
              label: "Configuration",
              slug: "reference/configuration",
            },
          ],
        },
      ],
    }),
  ],
});
