"use client";

import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent } from "@/components/ui/card";
import { Separator } from "@/components/ui/separator";
import { Message, Attachment } from "@/types/mail";
import { YourMailAPI } from "@/lib/api";
import { ComposeDialog } from "./ComposeDialog";
import {
  User as UserIcon,
  Reply,
  Download,
  Eye,
  Paperclip,
  Calendar,
  MessageCircle,
  FileText,
  Image,
  File,
  Video,
  Music,
} from "lucide-react";

interface MessageDetailViewProps {
  message: Message;
  api: YourMailAPI;
  onMessageSent: () => void;
}

export function MessageDetailView({
  message,
  api,
  onMessageSent,
}: MessageDetailViewProps) {
  const [showReplies, setShowReplies] = useState(true);

  const getFileIcon = (contentType: string) => {
    if (contentType.startsWith("image/")) return <Image className="h-4 w-4" />;
    if (contentType.startsWith("video/")) return <Video className="h-4 w-4" />;
    if (contentType.startsWith("audio/")) return <Music className="h-4 w-4" />;
    if (contentType.includes("pdf") || contentType.includes("document"))
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

  const handleDownload = async (attachment: Attachment) => {
    try {
      const blob = await api.downloadAttachment(attachment.id);
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = attachment.original_name;
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      URL.revokeObjectURL(url);
    } catch (error) {
      console.error("Failed to download attachment:", error);
    }
  };

  const handlePreview = (attachment: Attachment) => {
    // For images, videos, and PDFs, open in a new tab using the API URL
    if (
      attachment.content_type.startsWith("image/") ||
      attachment.content_type.startsWith("video/") ||
      attachment.content_type.includes("pdf")
    ) {
      window.open(api.getAttachmentUrl(attachment.id), "_blank");
    }
  };

  const canPreview = (contentType: string) => {
    return (
      contentType.startsWith("image/") ||
      contentType.startsWith("video/") ||
      contentType.includes("pdf") ||
      contentType.startsWith("text/")
    );
  };

  const MessageContent = ({
    msg,
    isReply = false,
  }: {
    msg: Message;
    isReply?: boolean;
  }) => (
    <Card className={`${isReply ? "ml-3 lg:ml-6 mt-4" : ""}`}>
      <CardContent className="p-4 lg:p-6">
        {/* Message Header */}
        <div className="space-y-3 lg:space-y-4">
          <div className="flex items-start justify-between gap-3">
            <div className="space-y-2 flex-1 min-w-0">
              <h3
                className={`font-semibold leading-tight ${
                  isReply ? "text-sm lg:text-base" : "text-base lg:text-lg"
                }`}
              >
                {isReply ? `Re: ${msg.subject}` : msg.subject || "(No subject)"}
              </h3>
              <div className="flex flex-col gap-2 text-xs lg:text-sm text-muted-foreground">
                <div className="flex items-center gap-2">
                  <UserIcon className="h-3 w-3 lg:h-4 lg:w-4 flex-shrink-0" />
                  <span className="truncate">From: {msg.from}</span>
                </div>
                <div className="flex items-center gap-2">
                  <Calendar className="h-3 w-3 lg:h-4 lg:w-4 flex-shrink-0" />
                  <span className="truncate">
                    {new Date(msg.timestamp).toLocaleString()}
                  </span>
                </div>
                {!msg.read && (
                  <Badge variant="secondary" className="text-xs w-fit">
                    New
                  </Badge>
                )}
              </div>
            </div>

            {!isReply && (
              <div className="flex items-center gap-2 flex-shrink-0">
                <ComposeDialog
                  api={api}
                  onMessageSent={onMessageSent}
                  replyTo={{
                    id: msg.id,
                    from: msg.from,
                    subject: msg.subject,
                    threadId: msg.thread_id,
                  }}
                />
              </div>
            )}
          </div>

          {/* Attachments */}
          {msg.attachments && msg.attachments.length > 0 && (
            <div className="space-y-2">
              <div className="flex items-center gap-2 text-sm font-medium">
                <Paperclip className="h-4 w-4" />
                {msg.attachments.length} Attachment
                {msg.attachments.length > 1 ? "s" : ""}
              </div>
              <div className="grid gap-2">
                {msg.attachments.map((attachment) => (
                  <div
                    key={attachment.id}
                    className="flex items-center justify-between p-3 border rounded-lg bg-muted/50"
                  >
                    <div className="flex items-center gap-3 flex-1 min-w-0">
                      {getFileIcon(attachment.content_type)}
                      <div className="min-w-0 flex-1">
                        <div className="font-medium text-sm truncate">
                          {attachment.original_name}
                        </div>
                        <div className="text-xs text-muted-foreground">
                          {formatFileSize(attachment.file_size)} •{" "}
                          {attachment.content_type}
                        </div>
                      </div>
                    </div>
                    <div className="flex items-center gap-1 lg:gap-2 flex-shrink-0">
                      {canPreview(attachment.content_type) && (
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => handlePreview(attachment)}
                          className="h-8 w-8 lg:h-9 lg:w-9"
                        >
                          <Eye className="h-3 w-3 lg:h-4 lg:w-4" />
                        </Button>
                      )}
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => handleDownload(attachment)}
                        className="h-8 w-8 lg:h-9 lg:w-9"
                      >
                        <Download className="h-3 w-3 lg:h-4 lg:w-4" />
                      </Button>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          )}

          <Separator />

          {/* Message Body */}
          <div className="space-y-4">
            {msg.is_html ? (
              <div
                className="prose prose-sm max-w-none dark:prose-invert prose-headings:text-foreground prose-p:text-foreground prose-strong:text-foreground prose-a:text-primary"
                dangerouslySetInnerHTML={{ __html: msg.body }}
              />
            ) : (
              <div className="whitespace-pre-wrap text-sm leading-relaxed break-words">
                {msg.body}
              </div>
            )}
          </div>
        </div>
      </CardContent>
    </Card>
  );

  return (
    <div className="h-full flex flex-col">
      <div className="flex-1 p-3 lg:p-6 overflow-auto space-y-4 lg:space-y-6">
        {/* Main Message */}
        <MessageContent msg={message} />

        {/* Threaded Replies */}
        {message.replies && message.replies.length > 0 && (
          <div className="space-y-4">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2">
                <MessageCircle className="h-4 w-4" />
                <span className="font-medium text-sm lg:text-base">
                  {message.replies.length} Repl
                  {message.replies.length === 1 ? "y" : "ies"}
                </span>
              </div>
              <Button
                variant="ghost"
                size="sm"
                onClick={() => setShowReplies(!showReplies)}
                className="text-xs lg:text-sm"
              >
                {showReplies ? "Hide" : "Show"} Replies
              </Button>
            </div>

            {showReplies && (
              <div className="space-y-4">
                {message.replies.map((reply) => (
                  <MessageContent key={reply.id} msg={reply} isReply />
                ))}
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  );
}
