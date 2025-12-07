import { defineCollection } from "astro:content";
import { docsSchema } from "@astrojs/starlight/schema";
import { glob } from "astro/loaders";
import path from "path";
import { fileURLToPath } from "url";

const __dirname = path.dirname(fileURLToPath(import.meta.url));

export const collections = {
  docs: defineCollection({
    loader: glob({
      pattern: "**/*.{md,mdx}",
      base: path.join(__dirname, "../../docs"),
    }),
    schema: docsSchema(),
  }),
};
