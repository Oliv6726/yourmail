"use client";

import { useEffect, useRef, useCallback, useState } from "react";
import { cn } from "@/lib/utils";

// Import Quill dynamically to avoid SSR issues
let Quill: any = null;
if (typeof window !== "undefined") {
  import("quill").then((quillModule) => {
    Quill = quillModule.default;
  });
}

interface RichTextEditorProps {
  value?: string;
  onChange?: (value: string) => void;
  placeholder?: string;
  className?: string;
  disabled?: boolean;
  minHeight?: number;
}

export function RichTextEditor({
  value = "",
  onChange,
  placeholder = "Start typing...",
  className,
  disabled = false,
  minHeight = 200,
}: RichTextEditorProps) {
  const editorRef = useRef<HTMLDivElement>(null);
  const quillRef = useRef<any>(null);
  const [isInitialized, setIsInitialized] = useState(false);
  const [isQuillLoaded, setIsQuillLoaded] = useState(false);

  const handleChange = useCallback(
    (content: string) => {
      onChange?.(content);
    },
    [onChange]
  );

  // Aggressive cleanup function
  const cleanup = useCallback(() => {
    if (quillRef.current) {
      try {
        // Remove all event listeners
        quillRef.current.off();

        // Get the container and completely clear it
        if (editorRef.current) {
          // Remove all Quill-generated elements
          const quillElements = editorRef.current.querySelectorAll(
            ".ql-toolbar, .ql-container, .ql-editor, .ql-tooltip"
          );
          quillElements.forEach((el) => el.remove());

          // Clear any remaining content
          editorRef.current.innerHTML = "";
        }

        // Clear the Quill reference
        quillRef.current = null;
      } catch (error) {
        console.warn("Error during Quill cleanup:", error);
      }
    }
    setIsInitialized(false);
  }, []);

  // Load Quill if not loaded
  useEffect(() => {
    if (!Quill && typeof window !== "undefined") {
      import("quill").then((quillModule) => {
        Quill = quillModule.default;
        setIsQuillLoaded(true);
      });
    } else if (Quill) {
      setIsQuillLoaded(true);
    }
  }, []);

  // Initialize Quill only once when ready
  useEffect(() => {
    if (!editorRef.current || !isQuillLoaded || !Quill || isInitialized) {
      return;
    }

    // Ensure we start with a clean slate
    cleanup();

    try {
      // Wait a tick to ensure cleanup is complete
      const timeoutId = setTimeout(() => {
        if (!editorRef.current || isInitialized) return;

        // Create a fresh container
        const container = document.createElement("div");
        editorRef.current.appendChild(container);

        // Quill configuration
        const toolbarOptions = [
          [{ header: [1, 2, 3, false] }],
          ["bold", "italic", "underline"],
          [{ color: [] }, { background: [] }],
          [{ list: "ordered" }, { list: "bullet" }],
          [{ align: [] }],
          ["link", "blockquote"],
          ["clean"],
        ];

        const quill = new Quill(container, {
          theme: "snow",
          placeholder,
          modules: {
            toolbar: toolbarOptions,
            clipboard: {
              matchVisual: false,
            },
          },
          formats: [
            "header",
            "bold",
            "italic",
            "underline",
            "color",
            "background",
            "list",
            "align",
            "link",
            "blockquote",
          ],
        });

        quillRef.current = quill;
        setIsInitialized(true);

        // Set initial content
        if (value) {
          quill.root.innerHTML = value;
        }

        // Handle content changes
        const handleTextChange = () => {
          if (quillRef.current) {
            const html = quillRef.current.root.innerHTML;
            if (html !== value) {
              handleChange(html);
            }
          }
        };

        quill.on("text-change", handleTextChange);

        // Set editor height
        const editor = container.querySelector(".ql-editor") as HTMLElement;
        if (editor) {
          editor.style.minHeight = `${minHeight}px`;
        }

        // Set disabled state
        quill.enable(!disabled);
      }, 10);

      return () => {
        clearTimeout(timeoutId);
      };
    } catch (error) {
      console.error("Failed to initialize Quill:", error);
      setIsInitialized(false);
    }
  }, [
    isQuillLoaded,
    placeholder,
    minHeight,
    handleChange,
    disabled,
    isInitialized,
    cleanup,
  ]);

  // Update content when value prop changes
  useEffect(() => {
    if (quillRef.current && isInitialized) {
      try {
        const currentContent = quillRef.current.root.innerHTML;
        if (value !== currentContent) {
          const selection = quillRef.current.getSelection();
          quillRef.current.root.innerHTML = value || "";
          if (selection) {
            try {
              quillRef.current.setSelection(selection);
            } catch (e) {
              // Ignore selection errors
            }
          }
        }
      } catch (error) {
        console.warn("Error updating Quill content:", error);
      }
    }
  }, [value, isInitialized]);

  // Update disabled state
  useEffect(() => {
    if (quillRef.current && isInitialized) {
      try {
        quillRef.current.enable(!disabled);
      } catch (error) {
        console.warn("Error updating Quill disabled state:", error);
      }
    }
  }, [disabled, isInitialized]);

  // Cleanup on unmount
  useEffect(() => {
    return cleanup;
  }, [cleanup]);

  return (
    <div className={cn("rich-text-editor", className)}>
      <div ref={editorRef} />
      <style jsx global>{`
        .ql-toolbar {
          border-top: 1px solid hsl(var(--border));
          border-left: 1px solid hsl(var(--border));
          border-right: 1px solid hsl(var(--border));
          border-bottom: none;
          border-radius: calc(var(--radius) - 2px) calc(var(--radius) - 2px) 0 0;
          background: hsl(var(--background));
          display: flex;
          flex-wrap: wrap;
          gap: 4px;
          padding: 8px;
        }

        .ql-toolbar .ql-formats {
          margin-right: 8px;
          display: flex;
          align-items: center;
          gap: 2px;
        }

        .ql-toolbar .ql-formats:last-child {
          margin-right: 0;
        }

        .ql-toolbar button {
          width: 28px !important;
          height: 28px !important;
          padding: 0 !important;
          border: none !important;
          border-radius: 4px !important;
          display: flex !important;
          align-items: center !important;
          justify-content: center !important;
        }

        .ql-toolbar .ql-picker {
          width: auto !important;
          min-width: 60px !important;
        }

        .ql-toolbar .ql-picker-label {
          padding: 4px 8px !important;
          border: none !important;
          border-radius: 4px !important;
          height: 28px !important;
          display: flex !important;
          align-items: center !important;
        }

        .ql-container {
          border-bottom: 1px solid hsl(var(--border));
          border-left: 1px solid hsl(var(--border));
          border-right: 1px solid hsl(var(--border));
          border-top: none;
          border-radius: 0 0 calc(var(--radius) - 2px) calc(var(--radius) - 2px);
          background: hsl(var(--background));
        }

        .ql-editor {
          color: hsl(var(--foreground));
          font-family: inherit;
          font-size: 14px;
          line-height: 1.5;
          padding: 12px 15px;
        }

        .ql-editor.ql-blank::before {
          color: hsl(var(--muted-foreground));
          font-style: normal;
          left: 15px;
        }

        .ql-toolbar .ql-stroke {
          stroke: hsl(var(--foreground));
        }

        .ql-toolbar .ql-fill {
          fill: hsl(var(--foreground));
        }

        .ql-toolbar .ql-picker-label {
          color: hsl(var(--foreground));
        }

        .ql-toolbar button:hover {
          background: hsl(var(--accent)) !important;
        }

        .ql-toolbar button.ql-active {
          background: hsl(var(--accent)) !important;
        }

        .ql-toolbar .ql-picker:hover .ql-picker-label {
          background: hsl(var(--accent)) !important;
        }

        .ql-toolbar .ql-picker-options {
          background: hsl(var(--background));
          border: 1px solid hsl(var(--border));
          border-radius: calc(var(--radius) - 2px);
          z-index: 1000;
        }

        .ql-toolbar .ql-picker-item:hover {
          background: hsl(var(--accent));
        }

        .ql-snow .ql-tooltip {
          background: hsl(var(--background));
          border: 1px solid hsl(var(--border));
          border-radius: calc(var(--radius) - 2px);
          color: hsl(var(--foreground));
          z-index: 1000;
        }

        .ql-snow .ql-tooltip input {
          background: hsl(var(--background));
          color: hsl(var(--foreground));
          border: 1px solid hsl(var(--border));
          border-radius: calc(var(--radius) - 2px);
          padding: 4px 8px;
        }

        .ql-snow .ql-tooltip a.ql-action:after {
          content: "Save";
        }

        .ql-snow .ql-tooltip a.ql-remove:after {
          content: "Remove";
        }

        /* Responsive toolbar */
        @media (max-width: 640px) {
          .ql-toolbar {
            padding: 6px;
            gap: 2px;
          }

          .ql-toolbar button {
            width: 24px !important;
            height: 24px !important;
          }

          .ql-toolbar .ql-picker-label {
            height: 24px !important;
            padding: 2px 6px !important;
            font-size: 12px !important;
          }
        }
      `}</style>
    </div>
  );
}
