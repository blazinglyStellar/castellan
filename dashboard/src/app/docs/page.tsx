import { ApiReference } from "@scalar/nextjs-api-reference"

const config = {
  url: process.env.NEXT_PUBLIC_API_URL
    ? `${process.env.NEXT_PUBLIC_API_URL}/openapi.yaml`
    : "/api/openapi.yaml",
  defaultHttpClient: "shell",
  hideDownloadButton: true,
}

export default function DocsPage() {
  return (
    <section>
      <ApiReference config={config} />
    </section>
  )
}
