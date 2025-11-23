import { loader } from "fumadocs-core/source"
import { createMDXSource } from "fumadocs-mdx"
import { structure } from "./static-source"

export function getSource() {
  return loader({
    baseUrl: "/docs",
    source: createMDXSource(structure, {
      schema: {
        frontmatter: {
          title: {
            type: "string",
            required: true,
          },
          description: {
            type: "string",
          },
          icon: {
            type: "string",
          },
        },
      },
    }),
  })
}
