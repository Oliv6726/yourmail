"use client";

import { useState } from "react";
import { Card, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Separator } from "@/components/ui/separator";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { Message } from "@/types/mail";
import { Mail, MailOpen, Clock, MessageCircle } from "lucide-react";

interface MessageListProps {
  messages: Message[];
  onMessageSelect?: (message: Message) => void;
  selectedMessageId?: number;
}

export function MessageList({
  messages,
  onMessageSelect,
  selectedMessageId,
}: MessageListProps) {
  const [expandedMessages, setExpandedMessages] = useState<Set<number>>(
    new Set()
  );

  const toggleExpanded = (messageId: number) => {
    setExpandedMessages((prev) => {
      const newSet = new Set(prev);
      if (newSet.has(messageId)) {
        newSet.delete(messageId);
      } else {
        newSet.add(messageId);
      }
      return newSet;
    });
  };

  const formatTimestamp = (timestamp: string) => {
    const date = new Date(timestamp);
    const now = new Date();
    const diffMs = now.getTime() - date.getTime();
    const diffMins = Math.floor(diffMs / 60000);
    const diffHours = Math.floor(diffMs / 3600000);
    const diffDays = Math.floor(diffMs / 86400000);

    if (diffMins < 1) return "Just now";
    if (diffMins < 60) return `${diffMins}m ago`;
    if (diffHours < 24) return `${diffHours}h ago`;
    if (diffDays < 7) return `${diffDays}d ago`;

    return date.toLocaleDateString();
  };

  const getInitials = (email: string) => {
    const [username] = email.split("@");
    return username.slice(0, 2).toUpperCase();
  };

  const isExpanded = (messageId: number) => expandedMessages.has(messageId);

  if (!messages || messages.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center h-full text-center p-8">
        <Mail className="h-16 w-16 text-muted-foreground mb-4" />
        <h3 className="text-lg font-semibold text-muted-foreground mb-2">
          No messages yet
        </h3>
        <p className="text-sm text-muted-foreground max-w-sm">
          Your inbox is empty. Start by composing a new message or wait for
          messages to arrive.
        </p>
      </div>
    );
  }

  return (
    <ScrollArea className="h-full">
      <div className="space-y-2 p-4">
        {messages.map((message, index) => (
          <div key={`${message.id}-${index}`}>
            <Card
              className={`cursor-pointer transition-all hover:shadow-md ${
                selectedMessageId === message.id ? "ring-2 ring-primary" : ""
              }`}
              onClick={() => {
                onMessageSelect?.(message);
                toggleExpanded(message.id);
              }}
            >
              <CardContent className="p-4">
                <div className="flex items-start gap-3">
                  <Avatar className="h-10 w-10 flex-shrink-0">
                    <AvatarFallback className="text-xs">
                      {getInitials(message.from)}
                    </AvatarFallback>
                  </Avatar>

                  <div className="flex-1 min-w-0">
                    <div className="flex items-center justify-between gap-2 mb-1">
                      <div className="flex items-center gap-2 min-w-0 flex-1">
                        <h4 className="font-semibold text-sm truncate">
                          {message.from}
                        </h4>
                        {!message.read && (
                          <Badge variant="secondary" className="text-xs">
                            New
                          </Badge>
                        )}
                        {message.replies && message.replies.length > 0 && (
                          <Badge
                            variant="outline"
                            className="text-xs flex items-center gap-1"
                          >
                            <MessageCircle className="h-3 w-3" />
                            {message.replies.length}
                          </Badge>
                        )}
                      </div>
                      <div className="flex items-center gap-2 text-xs text-muted-foreground flex-shrink-0">
                        <Clock className="h-3 w-3" />
                        {formatTimestamp(message.timestamp)}
                      </div>
                    </div>

                    <h5 className="font-medium text-sm mb-2 line-clamp-1">
                      {message.subject || "(No subject)"}
                    </h5>

                    <p
                      className={`text-sm text-muted-foreground ${
                        isExpanded(message.id) ? "" : "line-clamp-2"
                      }`}
                    >
                      {message.is_html
                        ? message.body
                            .replace(/<[^>]*>/g, "")
                            .substring(0, isExpanded(message.id) ? 500 : 100)
                        : message.body.substring(
                            0,
                            isExpanded(message.id) ? 500 : 100
                          )}
                      {message.body.length >
                        (isExpanded(message.id) ? 500 : 100) && "..."}
                    </p>

                    {message.body.length > 100 && (
                      <button
                        className="text-xs text-primary hover:underline mt-1"
                        onClick={(e) => {
                          e.stopPropagation();
                          toggleExpanded(message.id);
                        }}
                      >
                        {isExpanded(message.id) ? "Show less" : "Show more"}
                      </button>
                    )}
                  </div>

                  <div className="flex-shrink-0 ml-2">
                    {message.read ? (
                      <MailOpen className="h-4 w-4 text-muted-foreground" />
                    ) : (
                      <Mail className="h-4 w-4 text-primary" />
                    )}
                  </div>
                </div>
              </CardContent>
            </Card>

            {index < messages.length - 1 && <Separator className="my-2" />}
          </div>
        ))}
      </div>
    </ScrollArea>
  );
}
