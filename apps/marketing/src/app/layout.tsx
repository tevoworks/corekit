import type { Metadata } from "next"
import Link from "next/link"
import { Geist, Geist_Mono } from "next/font/google"
import "./globals.css"

const geistSans = Geist({ variable: "--font-geist-sans", subsets: ["latin"] })
const geistMono = Geist_Mono({ variable: "--font-geist-mono", subsets: ["latin"] })

export const metadata: Metadata = {
  title: { default: "CoolAdmin", template: "%s — CoolAdmin" },
  description: "A powerful admin panel and CMS platform for modern web applications.",
}

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en" className={`${geistSans.variable} ${geistMono.variable} scroll-smooth`}>
      <body className="min-h-screen flex flex-col font-sans antialiased bg-white text-zinc-900">
        <header className="sticky top-0 z-50 border-b border-zinc-200 bg-white/80 backdrop-blur-md">
          <div className="mx-auto flex h-16 max-w-7xl items-center justify-between px-4 sm:px-6 lg:px-8">
            <Link href="/" className="text-xl font-bold tracking-tight">
              CoolAdmin
            </Link>
            <nav className="hidden items-center gap-8 text-sm font-medium text-zinc-600 sm:flex">
              <Link href="/" className="hover:text-zinc-900 transition-colors">Home</Link>
              <Link href="/blog" className="hover:text-zinc-900 transition-colors">Blog</Link>
              <Link href="/contact" className="hover:text-zinc-900 transition-colors">Contact</Link>
            </nav>
            <div className="flex items-center gap-3">
              <a href="/login" className="text-sm font-medium text-zinc-600 hover:text-zinc-900 transition-colors">Log in</a>
              <a href="/register" className="inline-flex h-9 items-center justify-center rounded-lg bg-zinc-900 px-4 text-sm font-medium text-white hover:bg-zinc-800 transition-colors">
                Get Started
              </a>
            </div>
          </div>
        </header>

        <main className="flex-1">{children}</main>

        <footer className="border-t border-zinc-200 bg-zinc-50">
          <div className="mx-auto max-w-7xl px-4 py-12 sm:px-6 lg:px-8">
            <div className="flex flex-col items-center justify-between gap-4 sm:flex-row">
              <p className="text-sm text-zinc-500">© {new Date().getFullYear()} CoolAdmin. All rights reserved.</p>
              <div className="flex gap-6 text-sm text-zinc-500">
                <Link href="/blog" className="hover:text-zinc-900 transition-colors">Blog</Link>
                <Link href="/contact" className="hover:text-zinc-900 transition-colors">Contact</Link>
              </div>
            </div>
          </div>
        </footer>
      </body>
    </html>
  )
}
