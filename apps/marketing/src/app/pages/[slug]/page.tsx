import { apiFetch } from "@/lib/api"
import type { Page } from "@/lib/types"
import Image from "next/image"
import { notFound } from "next/navigation"

export const dynamic = "force-dynamic"

export default async function CMSPage({ params }: { params: Promise<{ slug: string }> }) {
  const { slug } = await params
  let page: Page
  try {
    page = await apiFetch<Page>(`/api/public/pages/${slug}`)
  } catch {
    notFound()
  }

  return (
    <div className="mx-auto max-w-3xl px-4 py-16 sm:px-6 lg:px-8">
      <h1 className="text-3xl font-bold tracking-tight text-zinc-900 sm:text-4xl">{page.title}</h1>
      {page.featured_image && (
        <Image src={page.featured_image} alt={page.title} width={1200} height={675} className="mt-8 w-full rounded-xl object-cover" />
      )}
      <div className="mt-8 prose prose-zinc max-w-none" dangerouslySetInnerHTML={{ __html: page.content }} />
    </div>
  )
}
