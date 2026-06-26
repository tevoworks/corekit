import { apiFetch } from "@/lib/api"
import type { Page } from "@/lib/types"

export const dynamic = "force-dynamic"

export default async function HomePage() {
  try {
    await apiFetch<Page>("/api/public/pages/landing")
  } catch {
    // Use static content
  }

  return (
    <div>
      <section className="relative overflow-hidden py-24 sm:py-32">
        <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
          <div className="mx-auto max-w-2xl text-center">
            <h1 className="text-4xl font-bold tracking-tight text-zinc-900 sm:text-6xl">
              Build something amazing
            </h1>
            <p className="mt-6 text-lg leading-8 text-zinc-600">
              A powerful platform to help you build, launch, and grow your product. Start your journey today.
            </p>
            <div className="mt-10 flex items-center justify-center gap-4">
              <a href="/register" className="rounded-lg bg-zinc-900 px-8 py-3 text-sm font-semibold text-white shadow-sm hover:bg-zinc-800 transition-colors">
                Get started
              </a>
              <a href="/contact" className="rounded-lg border border-zinc-300 px-8 py-3 text-sm font-semibold text-zinc-900 hover:bg-zinc-50 transition-colors">
                Contact sales
              </a>
            </div>
          </div>
        </div>
      </section>

      <section className="py-24 sm:py-32 bg-zinc-50">
        <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
          <div className="mx-auto max-w-2xl text-center mb-16">
            <h2 className="text-3xl font-bold tracking-tight text-zinc-900 sm:text-4xl">
              Everything you need
            </h2>
            <p className="mt-4 text-lg text-zinc-600">
              All the tools you need to build and scale your product in one place.
            </p>
          </div>
          <div className="grid grid-cols-1 gap-8 sm:grid-cols-2 lg:grid-cols-3">
            {[
              { title: "Content Management", desc: "Create and manage your content with our powerful CMS.", icon: "📝" },
              { title: "Blog Engine", desc: "Built-in blog with SEO-optimized pages and RSS feeds.", icon: "📰" },
              { title: "Contact Forms", desc: "Capture leads and manage customer inquiries effortlessly.", icon: "📬" },
              { title: "User Management", desc: "Full user authentication and profile management.", icon: "👥" },
              { title: "SEO Optimized", desc: "Server-side rendered pages for maximum search visibility.", icon: "🔍" },
              { title: "Analytics Ready", desc: "Track and measure your growth with built-in analytics.", icon: "📊" },
            ].map((f) => (
              <div key={f.title} className="rounded-xl border border-zinc-200 bg-white p-6">
                <div className="text-3xl mb-4">{f.icon}</div>
                <h3 className="text-lg font-semibold text-zinc-900">{f.title}</h3>
                <p className="mt-2 text-sm text-zinc-600">{f.desc}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      <section className="py-24 sm:py-32">
        <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
          <div className="mx-auto max-w-2xl text-center">
            <h2 className="text-3xl font-bold tracking-tight text-zinc-900 sm:text-4xl">
              Ready to get started?
            </h2>
            <p className="mt-4 text-lg text-zinc-600">
              Start building today. No credit card required.
            </p>
            <div className="mt-10">
              <a href="/register" className="rounded-lg bg-zinc-900 px-8 py-3 text-sm font-semibold text-white shadow-sm hover:bg-zinc-800 transition-colors">
                Get started free
              </a>
            </div>
          </div>
        </div>
      </section>
    </div>
  )
}
