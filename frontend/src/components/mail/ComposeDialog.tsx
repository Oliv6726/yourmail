"use client";

import { useState, useCallback, useEffect } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { Switch } from "@/components/ui/switch";
import { Badge } from "@/components/ui/badge";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
  SheetTrigger,
} from "@/components/ui/sheet";
import { RichTextEditor } from "@/components/ui/rich-text-editor";
import { FileUpload } from "@/components/ui/file-upload";
import { ComposeMessage } from "@/types/mail";
import { YourMailAPI } from "@/lib/api";
import {
  PenSquare,
  Send,
  X,
  Reply,
  Paperclip,
  Type,
  File,
  Image,
  Video,
  FileText,
} from "lucide-react";

interface ComposeDialogProps {
  api: YourMailAPI;
  onMessageSent?: () => void;
  replyTo?: {
    id: number;
    from: string;
    subject: string;
    threadId?: string;
  };
}

export function ComposeDialog({
  api,
  onMessageSent,
  replyTo,
}: ComposeDialogProps) {
  const [isOpen, setIsOpen] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  const [useRichText, setUseRichText] = useState(true);
  const [attachments, setAttachments] = useState<File[]>([]);
  const [dragActive, setDragActive] = useState(false);
  const [message, setMessage] = useState<ComposeMessage>({
    to: replyTo?.from || "",
    subject: replyTo?.subject ? `Re: ${replyTo.subject}` : "",
    body: "",
  });
  const [errors, setErrors] = useState<Record<string, string>>({});

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setErrors({});
    setIsLoading(true);

    // Validation
    const newErrors: Record<string, string> = {};
    if (!message.to.trim()) newErrors.to = "Recipient is required";
    if (!message.subject.trim()) newErrors.subject = "Subject is required";
    if (!message.body.trim()) newErrors.body = "Message body is required";

    // Validate email format
    if (message.to.trim() && !message.to.includes("@")) {
      newErrors.to = "Recipient must be in format: user@server";
    }

    if (Object.keys(newErrors).length > 0) {
      setErrors(newErrors);
      setIsLoading(false);
      return;
    }

    try {
      // Prepare message data
      const messageData = {
        to: message.to,
        subject: message.subject,
        body: message.body,
        is_html: useRichText,
        ...(replyTo?.threadId && { thread_id: replyTo.threadId }),
        ...(replyTo?.id && { parent_id: replyTo.id }),
      };

      // Send message - use file upload method if attachments exist
      if (attachments.length > 0) {
        await api.sendMessageWithFiles({
          ...messageData,
          attachments: attachments,
        });
      } else {
        await api.sendMessage(messageData);
      }

      // Reset form and close dialog
      setMessage({ to: "", subject: "", body: "" });
      setAttachments([]);
      setIsOpen(false);
      onMessageSent?.();

      // Show success (you could add a toast here)
      console.log("Message sent successfully!");
    } catch (error) {
      console.error("Failed to send message:", error);
      setErrors({
        general:
          error instanceof Error
            ? error.message
            : "Failed to send message. Please try again.",
      });
    } finally {
      setIsLoading(false);
    }
  };

  const handleInputChange = (field: keyof ComposeMessage, value: string) => {
    setMessage((prev) => ({ ...prev, [field]: value }));
    if (errors[field]) {
      setErrors((prev) => ({ ...prev, [field]: "" }));
    }
  };

  const handleFilesSelected = useCallback((files: File[]) => {
    setAttachments((prev) => [...prev, ...files]);
  }, []);

  const removeAttachment = useCallback((index: number) => {
    setAttachments((prev) => prev.filter((_, i) => i !== index));
  }, []);

  const handleClose = () => {
    setIsOpen(false);
    setMessage({ to: "", subject: "", body: "" });
    setAttachments([]);
    setErrors({});
    setUseRichText(true);
  };

  // Reset form when dialog opens/closes
  useEffect(() => {
    if (isOpen && replyTo) {
      setMessage({
        to: replyTo.from,
        subject: replyTo.subject ? `Re: ${replyTo.subject}` : "",
        body: "",
      });
    }
  }, [isOpen, replyTo]);

  // Drag and drop handlers for the entire dialog
  const handleDrag = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
  }, []);

  const handleDragIn = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    if (e.dataTransfer.items && e.dataTransfer.items.length > 0) {
      setDragActive(true);
    }
  }, []);

  const handleDragOut = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    setDragActive(false);
  }, []);

  const handleDrop = useCallback(
    (e: React.DragEvent) => {
      e.preventDefault();
      e.stopPropagation();
      setDragActive(false);

      if (e.dataTransfer.files && e.dataTransfer.files.length > 0) {
        const files = Array.from(e.dataTransfer.files);
        handleFilesSelected(files);
      }
    },
    [handleFilesSelected]
  );

  const getFileIcon = (file: File) => {
    if (file.type.startsWith("image/")) return <Image className="h-4 w-4" />;
    if (file.type.startsWith("video/")) return <Video className="h-4 w-4" />;
    if (file.type === "application/pdf")
      return <FileText className="h-4 w-4" />;
    return <File className="h-4 w-4" />;
  };

  const formatFileSize = (bytes: number) => {
    if (bytes === 0) return "0 Bytes";
    const k = 1024;
    const sizes = ["Bytes", "KB", "MB", "GB"];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + " " + sizes[i];
  };

  return (
    <Sheet open={isOpen} onOpenChange={setIsOpen}>
      <SheetTrigger asChild>
        <Button
          size={replyTo ? "sm" : "lg"}
          variant={replyTo ? "outline" : "default"}
          className={
            replyTo
              ? "w-full sm:w-auto"
              : "fixed bottom-6 right-6 rounded-full shadow-lg md:static md:rounded-md z-50"
          }
        >
          {replyTo ? (
            <Reply className="h-4 w-4 mr-2" />
          ) : (
            <PenSquare className="h-5 w-5 md:mr-2" />
          )}
          <span className={replyTo ? "inline" : "hidden md:inline"}>
            {replyTo ? "Reply" : "Compose"}
          </span>
        </Button>
      </SheetTrigger>
      <SheetContent
        side="bottom"
        className="h-[95vh] md:h-auto md:max-w-4xl md:mx-auto overflow-hidden"
        onDragEnter={handleDragIn}
        onDragLeave={handleDragOut}
        onDragOver={handleDrag}
        onDrop={handleDrop}
      >
        {/* Drag overlay */}
        {dragActive && (
          <div className="absolute inset-0 bg-primary/10 border-2 border-dashed border-primary z-50 flex items-center justify-center">
            <div className="text-center">
              <Paperclip className="h-12 w-12 mx-auto mb-4 text-primary" />
              <p className="text-lg font-medium text-primary">
                Drop files to attach
              </p>
            </div>
          </div>
        )}

        <div className="mx-auto max-w-4xl h-full flex flex-col">
          <SheetHeader className="flex-shrink-0">
            <div className="flex items-center justify-between">
              <div>
                <SheetTitle className="flex items-center gap-2">
                  {replyTo ? (
                    <>
                      <Reply className="h-5 w-5" />
                      Reply to Message
                    </>
                  ) : (
                    <>
                      <PenSquare className="h-5 w-5" />
                      Compose Message
                    </>
                  )}
                </SheetTitle>
                <SheetDescription>
                  {replyTo
                    ? `Replying to ${replyTo.from}`
                    : "Send a message to any user on the decentralized network"}
                </SheetDescription>
              </div>
              <Button variant="ghost" size="sm" onClick={handleClose}>
                <X className="h-4 w-4" />
              </Button>
            </div>
          </SheetHeader>

          <div className="flex-1 overflow-y-auto mt-6">
            <form
              onSubmit={handleSubmit}
              className="space-y-4 h-full flex flex-col"
            >
              {errors.general && (
                <div className="text-sm text-red-500 bg-red-50 dark:bg-red-950/20 p-3 rounded-md border border-red-200 dark:border-red-800">
                  {errors.general}
                </div>
              )}

              {/* Recipient and Subject */}
              <div className="grid gap-4 md:grid-cols-2">
                <div className="space-y-2">
                  <Label htmlFor="to">To</Label>
                  <Input
                    id="to"
                    type="email"
                    placeholder="bob@localhost"
                    value={message.to}
                    onChange={(e) => handleInputChange("to", e.target.value)}
                    className={errors.to ? "border-red-500" : ""}
                    disabled={isLoading}
                  />
                  {errors.to && (
                    <p className="text-sm text-red-500">{errors.to}</p>
                  )}
                </div>

                <div className="space-y-2">
                  <Label htmlFor="subject">Subject</Label>
                  <Input
                    id="subject"
                    type="text"
                    placeholder="Enter message subject"
                    value={message.subject}
                    onChange={(e) =>
                      handleInputChange("subject", e.target.value)
                    }
                    className={errors.subject ? "border-red-500" : ""}
                    disabled={isLoading}
                  />
                  {errors.subject && (
                    <p className="text-sm text-red-500">{errors.subject}</p>
                  )}
                </div>
              </div>

              {/* Rich Text Toggle */}
              <div className="flex items-center justify-between p-3 bg-muted/50 rounded-lg">
                <div className="flex items-center gap-2">
                  <Type className="h-4 w-4" />
                  <Label htmlFor="rich-text" className="font-medium">
                    Rich Text Editor
                  </Label>
                </div>
                <Switch
                  id="rich-text"
                  checked={useRichText}
                  onCheckedChange={setUseRichText}
                  disabled={isLoading}
                />
              </div>

              {/* Message Body */}
              <div className="space-y-2 flex-1 flex flex-col min-h-0">
                <Label htmlFor="body">Message</Label>
                <div className="flex-1 min-h-[200px]">
                  {useRichText ? (
                    <RichTextEditor
                      key={`rich-text-${isOpen}-${useRichText}-${Date.now()}`}
                      value={message.body}
                      onChange={(value) => handleInputChange("body", value)}
                      placeholder="Type your message here..."
                      className={`h-full ${
                        errors.body ? "border-red-500" : ""
                      }`}
                      disabled={isLoading}
                      minHeight={200}
                    />
                  ) : (
                    <Textarea
                      key={`plain-text-${isOpen}-${useRichText}`}
                      id="body"
                      placeholder="Type your message here..."
                      value={message.body}
                      onChange={(e) =>
                        handleInputChange("body", e.target.value)
                      }
                      className={`w-full h-full min-h-[200px] resize-none ${
                        errors.body ? "border-red-500" : ""
                      }`}
                      disabled={isLoading}
                    />
                  )}
                </div>
                {errors.body && (
                  <p className="text-sm text-red-500">{errors.body}</p>
                )}
              </div>

              {/* File Attachments */}
              <div className="space-y-3 flex-shrink-0">
                <div className="flex items-center justify-between">
                  <Label className="flex items-center gap-2">
                    <Paperclip className="h-4 w-4" />
                    Attachments
                  </Label>
                  <FileUpload
                    onFilesSelected={handleFilesSelected}
                    maxFiles={10}
                    maxFileSize={50 * 1024 * 1024}
                    disabled={isLoading}
                  />
                </div>

                {/* Attached Files List */}
                {attachments.length > 0 && (
                  <div className="space-y-2">
                    <div className="flex items-center gap-2">
                      <Badge variant="secondary" className="text-xs">
                        {attachments.length} file
                        {attachments.length !== 1 ? "s" : ""} attached
                      </Badge>
                    </div>
                    <div className="space-y-2">
                      {attachments.map((file, index) => (
                        <div
                          key={`${file.name}-${index}`}
                          className="flex items-center gap-3 p-2 bg-muted/30 rounded-lg"
                        >
                          <div className="flex-shrink-0">
                            {getFileIcon(file)}
                          </div>
                          <div className="flex-1 min-w-0">
                            <p className="text-sm font-medium truncate">
                              {file.name}
                            </p>
                            <p className="text-xs text-muted-foreground">
                              {formatFileSize(file.size)}
                            </p>
                          </div>
                          <Button
                            type="button"
                            variant="ghost"
                            size="sm"
                            onClick={() => removeAttachment(index)}
                            disabled={isLoading}
                          >
                            <X className="h-3 w-3" />
                          </Button>
                        </div>
                      ))}
                    </div>
                  </div>
                )}
              </div>

              {/* Actions */}
              <div className="flex flex-col sm:flex-row gap-2 pt-4 border-t flex-shrink-0">
                <Button
                  type="button"
                  variant="outline"
                  onClick={handleClose}
                  disabled={isLoading}
                  className="order-2 sm:order-1"
                >
                  Cancel
                </Button>
                <Button
                  type="submit"
                  disabled={isLoading}
                  className="order-1 sm:order-2 sm:ml-auto"
                >
                  {isLoading ? (
                    "Sending..."
                  ) : (
                    <>
                      <Send className="h-4 w-4 mr-2" />
                      Send Message
                      {attachments.length > 0 && (
                        <Badge variant="secondary" className="ml-2 text-xs">
                          {attachments.length}
                        </Badge>
                      )}
                    </>
                  )}
                </Button>
              </div>
            </form>
          </div>
        </div>
      </SheetContent>
    </Sheet>
  );
}
