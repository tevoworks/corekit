import type { MetadataRoute } from "next"

export default function robots(): MetadataRoute.Robots {
  return {
    rules: { userAgent: "*", allow: "/", disallow: ["/portal/", "/login", "/register"] },
    sitemap: `${process.env.SITE_URL || "http://localhost:3000"}/sitemap.xml`,
  }
}
