import { apiFetch } from "@/lib/api"
import type { BlogPost } from "@/lib/types"
import Link from "next/link"

export const revalidate = 300

export default async function BlogPage() {
  let posts: BlogPost[] = []
  try {
    posts = await apiFetch<BlogPost[]>("/api/public/blog")
  } catch {
    // API not available during build
  }
  return (
    <div className="mx-auto max-w-7xl px-4 py-16 sm:px-6 lg:px-8">
      <h1 className="text-3xl font-bold tracking-tight text-zinc-900 sm:text-4xl mb-8">Blog</h1>
      {posts.length === 0 ? (
        <p className="text-zinc-500">No posts yet.</p>
      ) : (
        <div className="grid gap-8 sm:grid-cols-2 lg:grid-cols-3">
          {posts.map((post) => (
            <Link key={post.id} href={`/blog/${post.slug}`} className="group rounded-xl border border-zinc-200 p-6 hover:border-zinc-300 transition-colors">
              <h2 className="text-lg font-semibold text-zinc-900 group-hover:text-zinc-600">{post.title}</h2>
              {post.excerpt && <p className="mt-2 text-sm text-zinc-600 line-clamp-3">{post.excerpt}</p>}
              <div className="mt-4 flex items-center gap-2 text-xs text-zinc-400">
                {post.published_at && <time>{new Date(post.published_at).toLocaleDateString()}</time>}
                {post.tags?.length > 0 && <span>· {post.tags.join(", ")}</span>}
              </div>
            </Link>
          ))}
        </div>
      )}
    </div>
  )
}
