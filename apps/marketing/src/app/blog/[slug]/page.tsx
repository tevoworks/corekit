import { apiFetch } from "@/lib/api"
import type { BlogPost } from "@/lib/types"
import { notFound } from "next/navigation"

export const revalidate = 300

export default async function BlogPostPage({ params }: { params: Promise<{ slug: string }> }) {
  const { slug } = await params
  let post: BlogPost
  try {
    post = await apiFetch<BlogPost>(`/api/public/blog/${slug}`)
  } catch {
    notFound()
  }

  return (
    <article className="mx-auto max-w-3xl px-4 py-16 sm:px-6 lg:px-8">
      <h1 className="text-3xl font-bold tracking-tight text-zinc-900 sm:text-4xl">{post.title}</h1>
      <div className="mt-4 flex items-center gap-2 text-sm text-zinc-500">
        {post.published_at && <time>{new Date(post.published_at).toLocaleDateString()}</time>}
        {post.tags?.length > 0 && <span>· {post.tags.join(", ")}</span>}
      </div>
      <div className="mt-8 prose prose-zinc max-w-none" dangerouslySetInnerHTML={{ __html: post.content }} />
    </article>
  )
}
