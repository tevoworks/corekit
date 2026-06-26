import { useEditor, EditorContent } from '@tiptap/react'
import StarterKit from '@tiptap/starter-kit'
import Link from '@tiptap/extension-link'
import Image from '@tiptap/extension-image'
import Underline from '@tiptap/extension-underline'
import TextAlign from '@tiptap/extension-text-align'
import { useState, useRef, useEffect } from 'react'
import MediaManager from './MediaManager'

interface RichTextEditorProps {
  content: string
  onChange: (html: string) => void
  placeholder?: string
  testId?: string
}

export default function RichTextEditor({ content, onChange, placeholder, testId }: RichTextEditorProps) {
  const [mediaOpen, setMediaOpen] = useState(false)
  const [sourceMode, setSourceMode] = useState(false)
  const [sourceHtml, setSourceHtml] = useState('')
  const sourceTextareaRef = useRef<HTMLTextAreaElement>(null)

  const editor = useEditor({
    extensions: [
      StarterKit.configure({
        heading: { levels: [1, 2, 3] },
      }),
      Link.configure({ openOnClick: false }),
      Image,
      Underline,
      TextAlign.configure({ types: ['heading', 'paragraph'] }),
    ],
    content,
    editorProps: {
      attributes: {
        class: 'prose prose-sm max-w-none focus:outline-none min-h-[200px] px-4 py-3',
      },
    },
    onUpdate: ({ editor }) => {
      if (!sourceMode) onChange(editor.getHTML())
    },
  })

  const toggleSource = () => {
    if (!editor) return
    if (!sourceMode) {
      setSourceHtml(editor.getHTML())
      setSourceMode(true)
    } else {
      editor.commands.setContent(sourceHtml)
      setSourceMode(false)
      onChange(sourceHtml)
    }
  }

  const handleSourceChange = (val: string) => {
    setSourceHtml(val)
    onChange(val)
  }

  useEffect(() => {
    if (sourceMode && sourceTextareaRef.current) {
      sourceTextareaRef.current.focus()
    }
  }, [sourceMode])

  if (!editor) return null

  const Btn = ({ onClick, active, label, title, testId }: {
    onClick: () => void; active?: boolean; label: string | React.ReactNode; title?: string; testId: string
  }) => (
    <button type="button" onClick={onClick} title={title}
      className={`px-1.5 py-1 text-xs font-medium rounded hover:bg-zinc-100 transition-colors leading-none whitespace-nowrap
        ${active ? 'bg-zinc-200 text-zinc-900' : 'text-zinc-600'}`}
      data-testid={testId}>
      {label}
    </button>
  )

  const Divider = () => <span className="w-px h-4 bg-zinc-300 mx-0.5 shrink-0" />

  return (
    <div className="border border-zinc-300 rounded-lg overflow-hidden" data-testid={testId}>
      <div className="flex flex-wrap items-center gap-0.5 border-b border-zinc-200 bg-zinc-50 px-2 py-1.5">
        <Btn onClick={() => editor.chain().focus().undo().run()} label="↶" title="Undo" testId="editor-undo" />
        <Btn onClick={() => editor.chain().focus().redo().run()} label="↷" title="Redo" testId="editor-redo" />

        <Divider />

        <Btn onClick={() => editor.chain().focus().toggleBold().run()} active={editor.isActive('bold')} label={<strong>B</strong>} title="Bold" testId="editor-bold" />
        <Btn onClick={() => editor.chain().focus().toggleItalic().run()} active={editor.isActive('italic')} label={<em>I</em>} title="Italic" testId="editor-italic" />
        <Btn onClick={() => editor.chain().focus().toggleUnderline().run()} active={editor.isActive('underline')} label={<span className="underline">U</span>} title="Underline" testId="editor-underline" />
        <Btn onClick={() => editor.chain().focus().toggleStrike().run()} active={editor.isActive('strike')} label={<span className="line-through">S</span>} title="Strikethrough" testId="editor-strike" />

        <Divider />

        <Btn onClick={() => editor.chain().focus().toggleHeading({ level: 1 }).run()} active={editor.isActive('heading', { level: 1 })} label="H1" title="Heading 1" testId="editor-h1" />
        <Btn onClick={() => editor.chain().focus().toggleHeading({ level: 2 }).run()} active={editor.isActive('heading', { level: 2 })} label="H2" title="Heading 2" testId="editor-h2" />
        <Btn onClick={() => editor.chain().focus().toggleHeading({ level: 3 }).run()} active={editor.isActive('heading', { level: 3 })} label="H3" title="Heading 3" testId="editor-h3" />

        <Divider />

        <Btn onClick={() => editor.chain().focus().toggleCode().run()} active={editor.isActive('code')} label="<>" title="Inline code" testId="editor-code" />
        <Btn onClick={() => editor.chain().focus().toggleCodeBlock().run()} active={editor.isActive('codeBlock')} label="Code" title="Code block" testId="editor-codeblock" />

        <Divider />

        <Btn onClick={() => editor.chain().focus().toggleBulletList().run()} active={editor.isActive('bulletList')} label="•" title="Bullet list" testId="editor-bullet-list" />
        <Btn onClick={() => editor.chain().focus().toggleOrderedList().run()} active={editor.isActive('orderedList')} label="1." title="Ordered list" testId="editor-ordered-list" />
        <Btn onClick={() => editor.chain().focus().toggleBlockquote().run()} active={editor.isActive('blockquote')} label="❝" title="Blockquote" testId="editor-quote" />

        <Divider />

        <Btn onClick={() => { const url = window.prompt('Link URL:'); if (url) editor.chain().focus().setLink({ href: url }).run() }}
          active={editor.isActive('link')} label="Link" title="Insert link" testId="editor-link" />
        <Btn onClick={() => setMediaOpen(true)} label="Img" title="Insert image" testId="editor-image" />

        <Divider />

        <Btn onClick={() => editor.chain().focus().setTextAlign('left').run()} active={editor.isActive({ textAlign: 'left' })} label="≡" title="Align left" testId="editor-align-left" />
        <Btn onClick={() => editor.chain().focus().setTextAlign('center').run()} active={editor.isActive({ textAlign: 'center' })} label="≡" title="Align center" testId="editor-align-center" />
        <Btn onClick={() => editor.chain().focus().setTextAlign('right').run()} active={editor.isActive({ textAlign: 'right' })} label="≡" title="Align right" testId="editor-align-right" />

        <Btn onClick={() => editor.chain().focus().setHorizontalRule().run()} label="—" title="Horizontal rule" testId="editor-hr" />

        <Btn onClick={() => editor.chain().focus().clearNodes().unsetAllMarks().run()} label="⌫" title="Clear formatting" testId="editor-clear" />

        <Divider />

        <Btn onClick={toggleSource} active={sourceMode} label="&lt;/&gt;" title="Toggle source editor" testId="editor-source" />
      </div>

      {sourceMode ? (
        <textarea
          ref={sourceTextareaRef}
          value={sourceHtml}
          onChange={e => handleSourceChange(e.target.value)}
          className="w-full min-h-[200px] p-4 text-sm font-mono bg-zinc-900 text-zinc-100 border-0 outline-none resize-y"
          data-testid="editor-source-textarea"
          placeholder="<p>HTML content...</p>"
          spellCheck={false}
        />
      ) : (
        <EditorContent editor={editor} />
      )}

      <MediaManager
        open={mediaOpen}
        onClose={() => setMediaOpen(false)}
        onSelect={(file) => { editor.chain().focus().setImage({ src: file.url }).run() }}
      />
    </div>
  )
}
