"use client";

import { useCallback, useRef } from "react";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { Paperclip, Upload } from "lucide-react";

interface FileUploadProps {
  onFilesSelected?: (files: File[]) => void;
  maxFiles?: number;
  maxFileSize?: number; // in bytes
  acceptedTypes?: string[];
  className?: string;
  disabled?: boolean;
  children?: React.ReactNode;
}

export function FileUpload({
  onFilesSelected,
  maxFiles = 10,
  maxFileSize = 50 * 1024 * 1024, // 50MB
  acceptedTypes = [
    "image/*",
    "video/*",
    "application/pdf",
    "application/msword",
    "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
    "application/vnd.ms-excel",
    "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
    "text/*",
  ],
  className,
  disabled = false,
  children,
}: FileUploadProps) {
  const fileInputRef = useRef<HTMLInputElement>(null);

  const handleFiles = useCallback(
    (newFiles: FileList | null) => {
      if (!newFiles || disabled) return;

      const validFiles: File[] = [];
      const fileArray = Array.from(newFiles);

      for (const file of fileArray) {
        // Check file count
        if (validFiles.length >= maxFiles) {
          alert(`Maximum ${maxFiles} files allowed`);
          break;
        }

        // Check file size
        if (file.size > maxFileSize) {
          alert(
            `File "${file.name}" is too large. Maximum size is ${Math.round(
              maxFileSize / 1024 / 1024
            )}MB`
          );
          continue;
        }

        // Check file type
        const isValidType = acceptedTypes.some((type) => {
          if (type.includes("*")) {
            const category = type.split("/")[0];
            return file.type.startsWith(category + "/");
          }
          return file.type === type;
        });

        if (!isValidType) {
          alert(`File type "${file.type}" is not supported`);
          continue;
        }

        validFiles.push(file);
      }

      if (validFiles.length > 0) {
        onFilesSelected?.(validFiles);
      }

      // Reset input
      if (fileInputRef.current) {
        fileInputRef.current.value = "";
      }
    },
    [maxFiles, maxFileSize, acceptedTypes, onFilesSelected, disabled]
  );

  const handleClick = () => {
    if (!disabled) {
      fileInputRef.current?.click();
    }
  };

  return (
    <div className={cn("relative", className)}>
      <Button
        type="button"
        variant="outline"
        size="sm"
        onClick={handleClick}
        disabled={disabled}
        className="w-full sm:w-auto"
      >
        <Paperclip className="h-4 w-4 mr-2" />
        {children || "Attach Files"}
      </Button>
      <input
        ref={fileInputRef}
        type="file"
        multiple
        accept={acceptedTypes.join(",")}
        onChange={(e) => handleFiles(e.target.files)}
        className="absolute inset-0 w-full h-full opacity-0 cursor-pointer pointer-events-none"
        disabled={disabled}
      />
    </div>
  );
}
