"use client";

import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible";
import { ComposeDialog } from "@/components/mail/ComposeDialog";
import { Message, Attachment } from "@/types/mail";
import { YourMailAPI } from "@/lib/api";
import {
  ChevronDown,
  ChevronRight,
  Reply,
  Paperclip,
  Download,
  Clock,
} from "lucide-react";
import { formatDistanceToNow } from "date-fns";

interface ThreadedMessageViewProps {
  message: Message;
  api: YourMailAPI;
  onMessageSent?: () => void;
}

export function ThreadedMessageView({
  message,
  api,
  onMessageSent,
}: ThreadedMessageViewProps) {
  const [expandedMessages, setExpandedMessages] = useState<Set<number>>(
    new Set([message.id])
  );

  const toggleMessage = (messageId: number) => {
    const newExpanded = new Set(expandedMessages);
    if (newExpanded.has(messageId)) {
      newExpanded.delete(messageId);
    } else {
      newExpanded.add(messageId);
    }
    setExpandedMessages(newExpanded);
  };

  const allMessages = [message, ...(message.replies || [])];
  const messageCount = allMessages.length;

  const formatTime = (timestamp: string) => {
    try {
      const date = new Date(timestamp);
      return formatDistanceToNow(date, { addSuffix: true });
    } catch {
      return timestamp;
    }
  };

  const downloadAttachment = async (attachment: Attachment) => {
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

  const formatFileSize = (bytes: number) => {
    if (bytes === 0) return "0 Bytes";
    const k = 1024;
    const sizes = ["Bytes", "KB", "MB", "GB"];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + " " + sizes[i];
  };

  return (
    <div className="space-y-4">
      {/* Thread header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <h3 className="font-semibold text-lg">{message.subject}</h3>
          {messageCount > 1 && (
            <Badge variant="secondary" className="text-xs">
              {messageCount} message{messageCount !== 1 ? "s" : ""}
            </Badge>
          )}
        </div>
        <ComposeDialog
          api={api}
          onMessageSent={onMessageSent}
          replyTo={{
            id: message.id,
            from: message.from,
            subject: message.subject,
            threadId: message.thread_id,
          }}
        />
      </div>

      {/* Message thread */}
      <div className="space-y-2">
        {allMessages.map((msg, index) => {
          const isExpanded = expandedMessages.has(msg.id);
          const isFirst = index === 0;
          const isLast = index === allMessages.length - 1;

          return (
            <Collapsible
              key={msg.id}
              open={isExpanded}
              onOpenChange={() => toggleMessage(msg.id)}
            >
              <div
                className={`border rounded-lg ${
                  isExpanded ? "border-primary/20 bg-muted/20" : "border-border"
                }`}
              >
                <CollapsibleTrigger asChild>
                  <div className="flex items-center justify-between p-4 cursor-pointer hover:bg-muted/50 transition-colors">
                    <div className="flex items-center gap-3 flex-1 min-w-0">
                      {/* Expand/collapse icon */}
                      {isExpanded ? (
                        <ChevronDown className="h-4 w-4 flex-shrink-0" />
                      ) : (
                        <ChevronRight className="h-4 w-4 flex-shrink-0" />
                      )}

                      {/* Message info */}
                      <div className="flex-1 min-w-0">
                        <div className="flex items-center gap-2 mb-1">
                          <span className="font-medium truncate">
                            {msg.from_user?.username || msg.from}
                          </span>
                          {!msg.read && (
                            <Badge variant="destructive" className="text-xs">
                              New
                            </Badge>
                          )}
                          {msg.attachments && msg.attachments.length > 0 && (
                            <Paperclip className="h-4 w-4 text-muted-foreground" />
                          )}
                        </div>
                        <div className="flex items-center gap-2 text-sm text-muted-foreground">
                          <Clock className="h-3 w-3" />
                          {formatTime(msg.timestamp)}
                          {!isExpanded && (
                            <span className="truncate ml-2">
                              {msg.is_html
                                ? msg.body
                                    .replace(/<[^>]*>/g, "")
                                    .substring(0, 100)
                                : msg.body.substring(0, 100)}
                              {msg.body.length > 100 && "..."}
                            </span>
                          )}
                        </div>
                      </div>
                    </div>

                    {/* Reply button for expanded messages */}
                    {isExpanded && (
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
                    )}
                  </div>
                </CollapsibleTrigger>

                <CollapsibleContent>
                  <div className="px-4 pb-4 border-t">
                    {/* Message content */}
                    <div className="mt-4">
                      {msg.is_html ? (
                        <div
                          className="prose prose-sm max-w-none dark:prose-invert"
                          dangerouslySetInnerHTML={{ __html: msg.body }}
                        />
                      ) : (
                        <div className="whitespace-pre-wrap text-sm">
                          {msg.body}
                        </div>
                      )}
                    </div>

                    {/* Attachments */}
                    {msg.attachments && msg.attachments.length > 0 && (
                      <div className="mt-4 pt-4 border-t">
                        <h4 className="text-sm font-medium mb-2 flex items-center gap-2">
                          <Paperclip className="h-4 w-4" />
                          Attachments ({msg.attachments.length})
                        </h4>
                        <div className="space-y-2">
                          {msg.attachments.map((attachment) => (
                            <div
                              key={attachment.id}
                              className="flex items-center justify-between p-2 bg-muted/50 rounded"
                            >
                              <div className="flex-1 min-w-0">
                                <p className="text-sm font-medium truncate">
                                  {attachment.original_name}
                                </p>
                                <p className="text-xs text-muted-foreground">
                                  {formatFileSize(attachment.file_size)} •{" "}
                                  {attachment.content_type}
                                </p>
                              </div>
                              <Button
                                variant="ghost"
                                size="sm"
                                onClick={() => downloadAttachment(attachment)}
                              >
                                <Download className="h-4 w-4" />
                              </Button>
                            </div>
                          ))}
                        </div>
                      </div>
                    )}
                  </div>
                </CollapsibleContent>
              </div>
            </Collapsible>
          );
        })}
      </div>
    </div>
  );
}
