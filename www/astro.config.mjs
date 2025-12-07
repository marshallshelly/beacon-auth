// @ts-check
import { defineConfig } from "astro/config";
import starlight from "@astrojs/starlight";

// https://astro.build/config
export default defineConfig({
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
      customCss: ["./src/styles/custom.css"],
      sidebar: [
        {
          label: "Start Here",
          items: [{ label: "Quickstart", slug: "getting-started/quickstart" }],
        },
        {
          label: "Guides",
          items: [{ label: "Plugins System", slug: "guides/plugins" }],
        },
        {
          label: "Reference",
          items: [{ label: "Configuration", slug: "reference/configuration" }],
        },
      ],
    }),
  ],
});
