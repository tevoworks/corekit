import { useEditor, EditorContent } from '@tiptap/react'
import StarterKit from '@tiptap/starter-kit'
import Link from '@tiptap/extension-link'
import Image from '@tiptap/extension-image'

interface RichTextEditorProps {
  content: string
  onChange: (html: string) => void
  placeholder?: string
  testId?: string
}

export default function RichTextEditor({ content, onChange, placeholder, testId }: RichTextEditorProps) {
  const editor = useEditor({
    extensions: [
      StarterKit,
      Link.configure({ openOnClick: false }),
      Image,
    ],
    content,
    editorProps: {
      attributes: {
        class: 'prose prose-sm max-w-none focus:outline-none min-h-[200px] px-4 py-3',
      },
    },
    onUpdate: ({ editor }) => {
      onChange(editor.getHTML())
    },
  })

  if (!editor) return null

  const ToolbarButton = ({ onClick, active, label, testId }: { onClick: () => void; active?: boolean; label: string; testId?: string }) => (
    <button
      type="button"
      onClick={onClick}
      className={`px-2 py-1 text-xs font-medium rounded hover:bg-zinc-100 transition-colors ${active ? 'bg-zinc-200 text-zinc-900' : 'text-zinc-600'}`}
      data-testid={testId}
    >
      {label}
    </button>
  )

  return (
    <div className="border border-zinc-300 rounded-lg overflow-hidden" data-testid={testId}>
      <div className="flex flex-wrap gap-1 border-b border-zinc-200 bg-zinc-50 px-3 py-2">
        <ToolbarButton onClick={() => editor.chain().focus().toggleBold().run()} active={editor.isActive('bold')} label="B" testId="editor-bold" />
        <ToolbarButton onClick={() => editor.chain().focus().toggleItalic().run()} active={editor.isActive('italic')} label="I" testId="editor-italic" />
        <ToolbarButton onClick={() => editor.chain().focus().toggleHeading({ level: 2 }).run()} active={editor.isActive('heading', { level: 2 })} label="H2" testId="editor-h2" />
        <ToolbarButton onClick={() => editor.chain().focus().toggleHeading({ level: 3 }).run()} active={editor.isActive('heading', { level: 3 })} label="H3" testId="editor-h3" />
        <ToolbarButton onClick={() => editor.chain().focus().toggleBulletList().run()} active={editor.isActive('bulletList')} label="• List" testId="editor-bullet-list" />
        <ToolbarButton onClick={() => editor.chain().focus().toggleOrderedList().run()} active={editor.isActive('orderedList')} label="1. List" testId="editor-ordered-list" />
        <ToolbarButton onClick={() => editor.chain().focus().toggleBlockquote().run()} active={editor.isActive('blockquote')} label="Quote" testId="editor-quote" />
        <ToolbarButton onClick={() => {
          const url = window.prompt('Link URL:')
          if (url) editor.chain().focus().setLink({ href: url }).run()
        }} active={editor.isActive('link')} label="Link" testId="editor-link" />
        <ToolbarButton onClick={() => {
          const url = window.prompt('Image URL:')
          if (url) editor.chain().focus().setImage({ src: url }).run()
        }} label="Img" testId="editor-image" />
      </div>
      <EditorContent editor={editor} />
    </div>
  )
}
