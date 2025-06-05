"use client";

import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
  SheetTrigger,
} from "@/components/ui/sheet";
import { ComposeMessage } from "@/types/mail";
import { YourMailAPI } from "@/lib/api";
import { PenSquare, Send, X } from "lucide-react";

interface ComposeDialogProps {
  api: YourMailAPI;
  onMessageSent?: () => void;
}

export function ComposeDialog({ api, onMessageSent }: ComposeDialogProps) {
  const [isOpen, setIsOpen] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  const [message, setMessage] = useState<ComposeMessage>({
    to: "",
    subject: "",
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
      // Send message via API
      await api.sendMessage({
        to: message.to,
        subject: message.subject,
        body: message.body,
      });

      // Reset form and close dialog
      setMessage({ to: "", subject: "", body: "" });
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

  const handleClose = () => {
    setIsOpen(false);
    setMessage({ to: "", subject: "", body: "" });
    setErrors({});
  };

  return (
    <Sheet open={isOpen} onOpenChange={setIsOpen}>
      <SheetTrigger asChild>
        <Button
          size="lg"
          className="fixed bottom-6 right-6 rounded-full shadow-lg md:static md:rounded-md"
        >
          <PenSquare className="h-5 w-5 md:mr-2" />
          <span className="hidden md:inline">Compose</span>
        </Button>
      </SheetTrigger>
      <SheetContent
        side="bottom"
        className="h-[90vh] md:h-auto md:max-w-2xl md:mx-auto"
      >
        <SheetHeader>
          <div className="flex items-center justify-between">
            <div>
              <SheetTitle>Compose Message</SheetTitle>
              <SheetDescription>
                Send a message to any user on the decentralized network
              </SheetDescription>
            </div>
            <Button variant="ghost" size="sm" onClick={handleClose}>
              <X className="h-4 w-4" />
            </Button>
          </div>
        </SheetHeader>

        <form
          onSubmit={handleSubmit}
          className="mt-6 space-y-4 h-full flex flex-col"
        >
          {errors.general && (
            <div className="text-sm text-red-500 bg-red-50 p-3 rounded-md">
              {errors.general}
            </div>
          )}

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
            {errors.to && <p className="text-sm text-red-500">{errors.to}</p>}
          </div>

          <div className="space-y-2">
            <Label htmlFor="subject">Subject</Label>
            <Input
              id="subject"
              type="text"
              placeholder="Enter message subject"
              value={message.subject}
              onChange={(e) => handleInputChange("subject", e.target.value)}
              className={errors.subject ? "border-red-500" : ""}
              disabled={isLoading}
            />
            {errors.subject && (
              <p className="text-sm text-red-500">{errors.subject}</p>
            )}
          </div>

          <div className="space-y-2 flex-1 flex flex-col">
            <Label htmlFor="body">Message</Label>
            <Textarea
              id="body"
              placeholder="Type your message here..."
              value={message.body}
              onChange={(e) => handleInputChange("body", e.target.value)}
              className={`flex-1 min-h-[200px] resize-none ${
                errors.body ? "border-red-500" : ""
              }`}
              disabled={isLoading}
            />
            {errors.body && (
              <p className="text-sm text-red-500">{errors.body}</p>
            )}
          </div>

          <div className="flex gap-2 pt-4">
            <Button
              type="button"
              variant="outline"
              onClick={handleClose}
              disabled={isLoading}
            >
              Cancel
            </Button>
            <Button type="submit" disabled={isLoading} className="flex-1">
              {isLoading ? (
                "Sending..."
              ) : (
                <>
                  <Send className="h-4 w-4 mr-2" />
                  Send Message
                </>
              )}
            </Button>
          </div>
        </form>
      </SheetContent>
    </Sheet>
  );
}
