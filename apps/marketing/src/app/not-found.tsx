import Link from "next/link"

export default function NotFound() {
  return (
    <div className="flex flex-col items-center justify-center py-24">
      <h1 className="text-4xl font-bold text-zinc-900">404</h1>
      <p className="mt-4 text-zinc-600">Page not found</p>
      <Link href="/" className="mt-8 rounded-lg bg-zinc-900 px-6 py-2.5 text-sm font-semibold text-white hover:bg-zinc-800 transition-colors">
        Go home
      </Link>
    </div>
  )
}
