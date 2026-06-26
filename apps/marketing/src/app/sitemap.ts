import type { MetadataRoute } from "next"

const API_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080"

export default async function sitemap(): Promise<MetadataRoute.Sitemap> {
  const baseUrl = process.env.SITE_URL || "http://localhost:3000"

  let blogEntries: MetadataRoute.Sitemap = []
  try {
    const res = await fetch(`${API_URL}/api/public/blog`)
    const json = await res.json()
    const posts: Array<{ slug: string; published_at?: string; updated_at: string }> = json.data || []
    blogEntries = posts.map((post) => ({
      url: `${baseUrl}/blog/${post.slug}`,
      lastModified: post.published_at || post.updated_at,
      changeFrequency: "weekly" as const,
      priority: 0.6,
    }))
  } catch {}

  return [
    { url: baseUrl, lastModified: new Date(), changeFrequency: "weekly", priority: 1 },
    { url: `${baseUrl}/blog`, lastModified: new Date(), changeFrequency: "daily", priority: 0.8 },
    { url: `${baseUrl}/contact`, lastModified: new Date(), changeFrequency: "monthly", priority: 0.5 },
    ...blogEntries,
  ]
}
