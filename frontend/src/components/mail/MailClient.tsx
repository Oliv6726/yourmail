"use client";

import { useState, useEffect, useCallback } from "react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Message } from "@/types/mail";
import { YourMailAPI, type User } from "@/lib/api";
import { MessageList } from "./MessageList";
import { ComposeDialog } from "./ComposeDialog";
import {
  Mail,
  RefreshCw,
  User as UserIcon,
  LogOut,
  Wifi,
  WifiOff,
  Bell,
  BellOff,
} from "lucide-react";

interface MailClientProps {
  user: User;
  api: YourMailAPI;
  onLogout: () => void;
}

export function MailClient({ user, api, onLogout }: MailClientProps) {
  const [messages, setMessages] = useState<Message[]>([]);
  const [selectedMessage, setSelectedMessage] = useState<Message | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [unreadCount, setUnreadCount] = useState(0);
  const [serverStatus, setServerStatus] = useState<
    "connected" | "disconnected" | "checking"
  >("checking");
  const [realtimeConnected, setRealtimeConnected] = useState(false);

  const handleRefresh = useCallback(async () => {
    setIsLoading(true);
    try {
      const result = await api.getMessages();
      // Handle null/undefined result gracefully
      const messagesArray = result || [];
      setMessages(messagesArray);
      console.log(`Loaded ${messagesArray.length} messages from server`);

      // Also refresh unread count
      const count = await api.getUnreadCount();
      setUnreadCount(count);
    } catch (error) {
      console.error("Failed to refresh messages:", error);
      // Keep existing messages if refresh fails
    } finally {
      setIsLoading(false);
    }
  }, [api]);

  // Check server health on mount and periodically
  useEffect(() => {
    const checkServerHealth = async () => {
      try {
        await api.healthCheck();
        setServerStatus("connected");
      } catch (error) {
        console.error("Server health check failed:", error);
        setServerStatus("disconnected");
      }
    };

    checkServerHealth();
    const interval = setInterval(checkServerHealth, 30000); // Check every 30 seconds

    return () => clearInterval(interval);
  }, [api]);

  // Set up real-time updates with Server-Sent Events
  useEffect(() => {
    const handleNewMessage = (message: Message) => {
      console.log("New message received via SSE:", message);
      setMessages((prev) => [message, ...prev]);

      // Show notification if available
      if ("Notification" in window && Notification.permission === "granted") {
        new Notification(`New message from ${message.from}`, {
          body: message.subject,
          icon: "/favicon.ico",
        });
      }
    };

    const handleUnreadCount = (count: number) => {
      console.log("Unread count updated via SSE:", count);
      setUnreadCount(count);
    };

    try {
      api.subscribeToInboxUpdates(handleNewMessage, handleUnreadCount);
      setRealtimeConnected(true);
      console.log("SSE connection established");
    } catch (error) {
      console.error("Failed to establish SSE connection:", error);
      setRealtimeConnected(false);
    }

    // Request notification permission
    if ("Notification" in window && Notification.permission === "default") {
      Notification.requestPermission();
    }

    // Cleanup SSE connection on unmount
    return () => {
      api.closeEventSource();
      setRealtimeConnected(false);
    };
  }, [api]);

  // Load messages on mount
  useEffect(() => {
    handleRefresh();
  }, [handleRefresh]);

  const handleMessageSent = () => {
    // Refresh messages after sending
    handleRefresh();
  };

  const handleMessageSelect = async (message: Message) => {
    setSelectedMessage(message);

    // Mark as read if it's unread
    if (!message.read) {
      try {
        await api.markAsRead(message.id);
        // Update local state
        setMessages((prev) =>
          prev.map((m) => (m.id === message.id ? { ...m, read: true } : m))
        );
        // Update unread count
        setUnreadCount((prev) => Math.max(0, prev - 1));
      } catch (error) {
        console.error("Failed to mark message as read:", error);
      }
    }
  };

  return (
    <div className="h-screen flex flex-col bg-background">
      {/* Header */}
      <div className="border-b bg-card">
        <div className="flex items-center justify-between p-4">
          <div className="flex items-center gap-3">
            <h1 className="text-xl font-bold flex items-center gap-2">
              <Mail className="h-6 w-6" />
              📬 YourMail
            </h1>
            <Badge variant="outline" className="hidden sm:flex">
              {user.username}
            </Badge>
          </div>

          <div className="flex items-center gap-2">
            {/* Real-time connection status */}
            <div className="flex items-center gap-1 text-sm">
              {realtimeConnected ? (
                <Bell className="h-4 w-4 text-green-500" />
              ) : (
                <BellOff className="h-4 w-4 text-orange-500" />
              )}
              <span className="hidden sm:inline text-muted-foreground">
                {realtimeConnected ? "Live" : "Offline"}
              </span>
            </div>

            {/* Server Status */}
            <div className="flex items-center gap-1 text-sm">
              {serverStatus === "connected" ? (
                <Wifi className="h-4 w-4 text-green-500" />
              ) : serverStatus === "disconnected" ? (
                <WifiOff className="h-4 w-4 text-red-500" />
              ) : (
                <RefreshCw className="h-4 w-4 animate-spin text-yellow-500" />
              )}
              <span className="hidden sm:inline text-muted-foreground">
                {serverStatus === "connected"
                  ? "Online"
                  : serverStatus === "disconnected"
                  ? "Offline"
                  : "Checking..."}
              </span>
            </div>

            <Button
              variant="ghost"
              size="sm"
              onClick={handleRefresh}
              disabled={isLoading}
            >
              <RefreshCw
                className={`h-4 w-4 ${isLoading ? "animate-spin" : ""}`}
              />
            </Button>

            <Button variant="ghost" size="sm" onClick={onLogout}>
              <LogOut className="h-4 w-4" />
            </Button>
          </div>
        </div>

        {/* Mobile user info */}
        <div className="sm:hidden px-4 pb-3">
          <div className="flex items-center justify-between">
            <Badge variant="outline">{user.username}</Badge>
            <div className="flex items-center gap-2 text-sm text-muted-foreground">
              <UserIcon className="h-3 w-3" />
              {user.email}
            </div>
          </div>
        </div>
      </div>

      {/* Main Content */}
      <div className="flex-1 flex flex-col lg:flex-row overflow-hidden">
        {/* Sidebar/Inbox List */}
        <div className="lg:w-96 lg:border-r flex flex-col">
          <div className="p-4 border-b bg-muted/30">
            <div className="flex items-center justify-between">
              <h2 className="font-semibold flex items-center gap-2">
                Inbox
                {unreadCount > 0 && (
                  <Badge variant="default" className="text-xs">
                    {unreadCount}
                  </Badge>
                )}
              </h2>
              <div className="hidden lg:block">
                <ComposeDialog api={api} onMessageSent={handleMessageSent} />
              </div>
            </div>
          </div>

          <div className="flex-1 overflow-hidden">
            <MessageList
              messages={messages}
              onMessageSelect={handleMessageSelect}
              selectedMessageId={selectedMessage?.id}
            />
          </div>
        </div>

        {/* Message Detail View (Desktop) */}
        <div className="hidden lg:flex flex-1 flex-col">
          {selectedMessage ? (
            <div className="h-full flex flex-col">
              <div className="p-6 border-b bg-muted/30">
                <h3 className="text-lg font-semibold mb-2">
                  {selectedMessage.subject || "(No subject)"}
                </h3>
                <div className="flex items-center gap-4 text-sm text-muted-foreground">
                  <div className="flex items-center gap-2">
                    <UserIcon className="h-4 w-4" />
                    <span>From: {selectedMessage.from}</span>
                  </div>
                  <div>
                    {new Date(selectedMessage.timestamp).toLocaleString()}
                  </div>
                  {!selectedMessage.read && (
                    <Badge variant="secondary" className="text-xs">
                      New
                    </Badge>
                  )}
                </div>
              </div>

              <div className="flex-1 p-6 overflow-auto">
                <div className="whitespace-pre-wrap text-sm leading-relaxed">
                  {selectedMessage.body}
                </div>
              </div>
            </div>
          ) : (
            <div className="flex-1 flex items-center justify-center">
              <div className="text-center text-muted-foreground">
                <Mail className="h-16 w-16 mx-auto mb-4 opacity-50" />
                <h3 className="text-lg font-medium mb-2">
                  No message selected
                </h3>
                <p className="text-sm">
                  Select a message from your inbox to read it
                </p>
              </div>
            </div>
          )}
        </div>
      </div>

      {/* Mobile Compose Button */}
      <div className="lg:hidden p-4 border-t">
        <ComposeDialog api={api} onMessageSent={handleMessageSent} />
      </div>
    </div>
  );
}
