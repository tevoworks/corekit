import { useEditor, EditorContent } from '@tiptap/react'
import { BubbleMenu } from '@tiptap/extension-bubble-menu'
import StarterKit from '@tiptap/starter-kit'
import Link from '@tiptap/extension-link'
import Image from '@tiptap/extension-image'
import Underline from '@tiptap/extension-underline'
import TextAlign from '@tiptap/extension-text-align'
import { Table } from '@tiptap/extension-table'
import { TableRow } from '@tiptap/extension-table-row'
import { TableCell } from '@tiptap/extension-table-cell'
import { TableHeader } from '@tiptap/extension-table-header'
import { TextStyle } from '@tiptap/extension-text-style'
import { Color } from '@tiptap/extension-color'
import { Highlight } from '@tiptap/extension-highlight'
import { TaskList } from '@tiptap/extension-task-list'
import { TaskItem } from '@tiptap/extension-task-item'
import { Subscript } from '@tiptap/extension-subscript'
import { Superscript } from '@tiptap/extension-superscript'
import { CharacterCount } from '@tiptap/extension-character-count'
import { useState, useRef, useEffect, useCallback } from 'react'
import MediaManager from './MediaManager'

interface RichTextEditorProps {
  content: string
  onChange: (html: string) => void
  placeholder?: string
  testId?: string
}

const COLORS = [
  { label: 'Default', value: '' },
  { label: 'Gray', value: '#6b7280' },
  { label: 'Red', value: '#ef4444' },
  { label: 'Orange', value: '#f97316' },
  { label: 'Amber', value: '#f59e0b' },
  { label: 'Green', value: '#22c55e' },
  { label: 'Blue', value: '#3b82f6' },
  { label: 'Purple', value: '#a855f7' },
  { label: 'Pink', value: '#ec4899' },
]

export default function RichTextEditor({ content, onChange, placeholder, testId }: RichTextEditorProps) {
  const [mediaOpen, setMediaOpen] = useState(false)
  const [sourceMode, setSourceMode] = useState(false)
  const [sourceHtml, setSourceHtml] = useState('')
  const [tableRows, setTableRows] = useState(3)
  const [tableCols, setTableCols] = useState(3)
  const [showTableInput, setShowTableInput] = useState(false)
  const [imageWidth, setImageWidth] = useState('')
  const sourceTextareaRef = useRef<HTMLTextAreaElement>(null)

  const editor = useEditor({
    extensions: [
      StarterKit.configure({
        heading: { levels: [1, 2, 3] },
        typography: true,
        gapcursor: true,
        dropcursor: true,
      }),
      Link.configure({ openOnClick: false }),
      Image,
      Underline,
      TextAlign.configure({ types: ['heading', 'paragraph', 'image'] }),
      Table.configure({ resizable: true }),
      TableRow,
      TableCell,
      TableHeader,
      TextStyle,
      Color,
      Highlight.configure({ multicolor: true }),
      TaskList,
      TaskItem.configure({ nested: true }),
      Subscript,
      Superscript,
      CharacterCount.configure({ limit: 10000 }),
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

  useEffect(() => {
    if (!editor || showTableInput) return
  }, [editor, showTableInput])

  const insertTable = useCallback(() => {
    if (!editor) return
    editor.chain().focus().insertTable({ rows: tableRows, cols: tableCols, withHeaderRow: true }).run()
    setShowTableInput(false)
  }, [editor, tableRows, tableCols])

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

  const ColorBtn = ({ label, title, testId }: { label: string; title?: string; testId: string }) => (
    <div className="relative group">
      <button type="button" title={title}
        className="px-1.5 py-1 text-xs font-medium rounded hover:bg-zinc-100 transition-colors leading-none text-zinc-600"
        data-testid={testId}>
        {label}
      </button>
      <div className="absolute top-full left-0 mt-1 p-1.5 bg-white border border-zinc-200 rounded-lg shadow-lg hidden group-hover:grid grid-cols-5 gap-1 z-50 min-w-[120px]">
        {COLORS.map(c => (
          <button key={c.value || 'default'}
            onClick={() => {
              if (testId === 'editor-text-color') {
                editor.chain().focus().setColor(c.value).run()
              } else {
                c.value ? editor.chain().focus().toggleHighlight({ color: c.value }).run() : editor.chain().focus().unsetHighlight().run()
              }
            }}
            className="w-5 h-5 rounded border border-zinc-200 hover:scale-110 transition-transform"
            style={{ backgroundColor: c.value || '#fff' }}
            title={c.label}
            data-testid={`${testId}-${c.label.toLowerCase()}`}
          />
        ))}
      </div>
    </div>
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
        <Btn onClick={() => editor.chain().focus().toggleTaskList().run()} active={editor.isActive('taskList')} label="☑" title="Task list" testId="editor-task-list" />
        <Btn onClick={() => editor.chain().focus().toggleBlockquote().run()} active={editor.isActive('blockquote')} label="❝" title="Blockquote" testId="editor-quote" />

        <Divider />

        <Btn onClick={() => { const url = window.prompt('Link URL:'); if (url) editor.chain().focus().setLink({ href: url }).run() }}
          active={editor.isActive('link')} label="Link" title="Insert link" testId="editor-link" />
        <Btn onClick={() => setMediaOpen(true)} label="Img" title="Insert image" testId="editor-image" />
        <Btn onClick={() => setShowTableInput(true)} label="Tbl" title="Insert table" testId="editor-table" />

        <Divider />

        <Btn onClick={() => editor.chain().focus().toggleSubscript().run()} active={editor.isActive('subscript')} label="x₂" title="Subscript" testId="editor-subscript" />
        <Btn onClick={() => editor.chain().focus().toggleSuperscript().run()} active={editor.isActive('superscript')} label="x²" title="Superscript" testId="editor-superscript" />

        <Divider />

        <ColorBtn label="A" title="Text color" testId="editor-text-color" />
        <ColorBtn label="H" title="Highlight color" testId="editor-highlight" />

        <Divider />

        <Btn onClick={() => editor.chain().focus().setTextAlign('left').run()} active={editor.isActive({ textAlign: 'left' })} label="≡" title="Align left" testId="editor-align-left" />
        <Btn onClick={() => editor.chain().focus().setTextAlign('center').run()} active={editor.isActive({ textAlign: 'center' })} label="≡" title="Align center" testId="editor-align-center" />
        <Btn onClick={() => editor.chain().focus().setTextAlign('right').run()} active={editor.isActive({ textAlign: 'right' })} label="≡" title="Align right" testId="editor-align-right" />

        <Btn onClick={() => editor.chain().focus().setHorizontalRule().run()} label="—" title="Horizontal rule" testId="editor-hr" />

        <Btn onClick={() => editor.chain().focus().clearNodes().unsetAllMarks().run()} label="⌫" title="Clear formatting" testId="editor-clear" />

        <Divider />

        <Btn onClick={toggleSource} active={sourceMode} label="&lt;/&gt;" title="Toggle source editor" testId="editor-source" />
      </div>

      {showTableInput && (
        <div className="flex items-center gap-2 px-3 py-2 border-b border-zinc-200 bg-white" data-testid="editor-table-input">
          <span className="text-xs text-zinc-600">Rows:</span>
          <input type="number" min={1} max={20} value={tableRows} onChange={e => setTableRows(Number(e.target.value))}
            className="w-14 px-2 py-1 text-xs border border-zinc-300 rounded" data-testid="editor-table-rows" />
          <span className="text-xs text-zinc-600">Cols:</span>
          <input type="number" min={1} max={20} value={tableCols} onChange={e => setTableCols(Number(e.target.value))}
            className="w-14 px-2 py-1 text-xs border border-zinc-300 rounded" data-testid="editor-table-cols" />
          <Btn onClick={insertTable} label="Insert" testId="editor-table-insert" />
          <Btn onClick={() => setShowTableInput(false)} label="Cancel" testId="editor-table-cancel" />
        </div>
      )}

      {editor && (
        <BubbleMenu editor={editor} tippyOptions={{ duration: 100 }}
          shouldShow={({ editor, state }) => {
            const { selection } = state
            const { $anchor } = selection
            const node = $anchor.node()
            const nodeType = node.type.name
            const isTable = nodeType === 'table' || nodeType === 'tableRow' || nodeType === 'tableCell' || nodeType === 'tableHeader'
            return isTable
          }}
          data-testid="editor-table-bubble"
        >
          <div className="flex items-center gap-0.5 bg-white border border-zinc-200 rounded-lg shadow-lg px-1 py-0.5 text-xs">
            <Btn onClick={() => editor.chain().focus().addRowBefore().run()} label="Row ↑" testId="editor-table-add-row-before" />
            <Btn onClick={() => editor.chain().focus().addRowAfter().run()} label="Row ↓" testId="editor-table-add-row-after" />
            <Btn onClick={() => editor.chain().focus().addColumnBefore().run()} label="Col ←" testId="editor-table-add-col-before" />
            <Btn onClick={() => editor.chain().focus().addColumnAfter().run()} label="Col →" testId="editor-table-add-col-after" />
            <Divider />
            <Btn onClick={() => editor.chain().focus().deleteRow().run()} label="Del Row" testId="editor-table-del-row" />
            <Btn onClick={() => editor.chain().focus().deleteColumn().run()} label="Del Col" testId="editor-table-del-col" />
            <Btn onClick={() => editor.chain().focus().deleteTable().run()} label="Del Tbl" testId="editor-table-del-table" />
          </div>
        </BubbleMenu>
      )}

      {editor && (
        <BubbleMenu editor={editor} tippyOptions={{ duration: 100 }}
          shouldShow={({ editor }) => editor.isActive('image')}
          data-testid="editor-image-bubble"
        >
          <div className="flex items-center gap-1.5 bg-white border border-zinc-200 rounded-lg shadow-lg px-2 py-1.5 text-xs">
            <span className="text-zinc-500">W:</span>
            <input type="number" value={imageWidth} onChange={e => {
              setImageWidth(e.target.value)
              const val = e.target.value ? `${e.target.value}px` : ''
              editor.chain().focus().updateAttributes('image', { width: val, style: val ? `width: ${val}` : '' }).run()
            }} className="w-14 px-1 py-0.5 border border-zinc-300 rounded text-xs" placeholder="auto" data-testid="editor-image-width" />
            <Divider />
            <Btn onClick={() => editor.chain().focus().setTextAlign('left').run()} active={editor.isActive({ textAlign: 'left' })} label="≡" testId="editor-image-align-left" />
            <Btn onClick={() => editor.chain().focus().setTextAlign('center').run()} active={editor.isActive({ textAlign: 'center' })} label="≡" testId="editor-image-align-center" />
            <Btn onClick={() => editor.chain().focus().setTextAlign('right').run()} active={editor.isActive({ textAlign: 'right' })} label="≡" testId="editor-image-align-right" />
            <Divider />
            <Btn onClick={() => editor.chain().focus().deleteSelection().run()} label="🗑" testId="editor-image-delete" />
          </div>
        </BubbleMenu>
      )}

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

      <div className="flex items-center justify-between px-3 py-1.5 border-t border-zinc-200 bg-zinc-50 text-[10px] text-zinc-400" data-testid="editor-char-count">
        <span>
          {editor.storage.characterCount?.words?.() || 0} words | {editor.storage.characterCount?.characters?.() || 0} characters
        </span>
        <span>
          {10000 - (editor.storage.characterCount?.characters?.() || 0)} remaining
        </span>
      </div>

      <MediaManager
        open={mediaOpen}
        onClose={() => setMediaOpen(false)}
        onSelect={(file) => { editor.chain().focus().setImage({ src: file.url }).run() }}
      />
    </div>
  )
}
