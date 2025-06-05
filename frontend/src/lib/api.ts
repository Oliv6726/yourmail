import { Message, Attachment } from "@/types/mail";

export interface User {
  id: number;
  username: string;
  email: string;
  created_at: string;
  updated_at: string;
}

export interface LoginResponse {
  success: boolean;
  message: string;
  token?: string;
  user?: User;
}

export interface RegisterRequest {
  username: string;
  email: string;
  password: string;
}

export interface SendMessageRequest {
  to: string;
  subject: string;
  body: string;
  is_html?: boolean;
  thread_id?: string;
  parent_id?: number;
}

export interface SendMessageWithFilesRequest extends SendMessageRequest {
  attachments?: File[];
}

export class YourMailAPI {
  private baseUrl: string;
  private token: string | null = null;
  private eventSource: EventSource | null = null;

  constructor(host: string = "localhost", port: number = 8080) {
    this.baseUrl = `http://${host}:${port}`;

    // Try to load saved token from localStorage
    if (typeof window !== "undefined") {
      this.token = localStorage.getItem("yourmail_token");
    }
  }

  // Set JWT token
  setToken(token: string) {
    this.token = token;
    if (typeof window !== "undefined") {
      localStorage.setItem("yourmail_token", token);
    }
  }

  // Clear JWT token
  clearToken() {
    this.token = null;
    if (typeof window !== "undefined") {
      localStorage.removeItem("yourmail_token");
    }
    this.closeEventSource();
  }

  // Get current token
  getToken(): string | null {
    return this.token;
  }

  // Check if user is authenticated
  isAuthenticated(): boolean {
    return !!this.token;
  }

  // Get authorization headers
  private getAuthHeaders(): Record<string, string> {
    const headers: Record<string, string> = {
      "Content-Type": "application/json",
    };

    if (this.token) {
      headers["Authorization"] = `Bearer ${this.token}`;
    }

    return headers;
  }

  // Register a new user
  async register(userData: RegisterRequest): Promise<LoginResponse> {
    const response = await fetch(`${this.baseUrl}/api/register`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify(userData),
    });

    const data = await response.json();

    if (data.success && data.token) {
      this.setToken(data.token);
    }

    return data;
  }

  // Login with username and password
  async login(username: string, password: string): Promise<LoginResponse> {
    const response = await fetch(`${this.baseUrl}/api/login`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({ username, password }),
    });

    const data = await response.json();

    if (data.success && data.token) {
      this.setToken(data.token);
    }

    return data;
  }

  // Logout
  async logout(): Promise<void> {
    this.clearToken();
  }

  // Get current user profile
  async getProfile(): Promise<User> {
    if (!this.token) {
      throw new Error("Not authenticated. Please login first.");
    }

    const response = await fetch(`${this.baseUrl}/api/profile`, {
      headers: this.getAuthHeaders(),
    });

    if (!response.ok) {
      if (response.status === 401) {
        this.clearToken();
        throw new Error("Authentication expired. Please login again.");
      }
      throw new Error(`Failed to fetch profile: ${response.status}`);
    }

    return await response.json();
  }

  // Get user's inbox messages
  async getMessages(
    limit: number = 50,
    offset: number = 0
  ): Promise<Message[]> {
    if (!this.token) {
      throw new Error("Not authenticated. Please login first.");
    }

    const params = new URLSearchParams({
      limit: limit.toString(),
      offset: offset.toString(),
    });

    const response = await fetch(`${this.baseUrl}/api/messages?${params}`, {
      headers: this.getAuthHeaders(),
    });

    if (!response.ok) {
      if (response.status === 401) {
        this.clearToken();
        throw new Error("Authentication expired. Please login again.");
      }
      throw new Error(`Failed to fetch messages: ${response.status}`);
    }

    return await response.json();
  }

  // Get user's sent messages
  async getSentMessages(
    limit: number = 50,
    offset: number = 0
  ): Promise<Message[]> {
    if (!this.token) {
      throw new Error("Not authenticated. Please login first.");
    }

    const params = new URLSearchParams({
      limit: limit.toString(),
      offset: offset.toString(),
    });

    const response = await fetch(
      `${this.baseUrl}/api/messages/sent?${params}`,
      {
        headers: this.getAuthHeaders(),
      }
    );

    if (!response.ok) {
      if (response.status === 401) {
        this.clearToken();
        throw new Error("Authentication expired. Please login again.");
      }
      throw new Error(`Failed to fetch sent messages: ${response.status}`);
    }

    return await response.json();
  }

  // Get unread message count
  async getUnreadCount(): Promise<number> {
    if (!this.token) {
      throw new Error("Not authenticated. Please login first.");
    }

    const response = await fetch(`${this.baseUrl}/api/messages/unread-count`, {
      headers: this.getAuthHeaders(),
    });

    if (!response.ok) {
      if (response.status === 401) {
        this.clearToken();
        throw new Error("Authentication expired. Please login again.");
      }
      throw new Error(`Failed to fetch unread count: ${response.status}`);
    }

    const data = await response.json();
    return data.unread_count;
  }

  // Mark message as read
  async markAsRead(messageId: number): Promise<void> {
    if (!this.token) {
      throw new Error("Not authenticated. Please login first.");
    }

    const response = await fetch(
      `${this.baseUrl}/api/messages/${messageId}/read`,
      {
        method: "POST",
        headers: this.getAuthHeaders(),
      }
    );

    if (!response.ok) {
      if (response.status === 401) {
        this.clearToken();
        throw new Error("Authentication expired. Please login again.");
      }
      throw new Error(`Failed to mark message as read: ${response.status}`);
    }
  }

  // Send a message
  async sendMessage(messageData: SendMessageRequest): Promise<{
    success: boolean;
    message: string;
    id?: number;
  }> {
    if (!this.token) {
      throw new Error("Not authenticated. Please login first.");
    }

    const response = await fetch(`${this.baseUrl}/api/send`, {
      method: "POST",
      headers: this.getAuthHeaders(),
      body: JSON.stringify(messageData),
    });

    if (!response.ok) {
      if (response.status === 401) {
        this.clearToken();
        throw new Error("Authentication expired. Please login again.");
      }

      // Read response body once and try to parse as JSON
      const responseText = await response.text();
      try {
        const errorData = JSON.parse(responseText);
        if (errorData.message) {
          throw new Error(`Failed to send message: ${errorData.message}`);
        } else if (errorData.error) {
          throw new Error(`Failed to send message: ${errorData.error}`);
        } else {
          throw new Error(
            `Failed to send message: ${JSON.stringify(errorData)}`
          );
        }
      } catch (jsonError) {
        // If JSON parsing fails, use the raw text
        throw new Error(`Failed to send message: ${responseText}`);
      }
    }

    return await response.json();
  }

  // Send a message with file attachments
  async sendMessageWithFiles(
    messageData: SendMessageWithFilesRequest
  ): Promise<{
    success: boolean;
    message: string;
    id?: number;
  }> {
    if (!this.token) {
      throw new Error("Not authenticated. Please login first.");
    }

    // Create FormData for multipart upload
    const formData = new FormData();
    formData.append("to", messageData.to);
    formData.append("subject", messageData.subject);
    formData.append("body", messageData.body);

    // Properly convert boolean to string for form data, default to false if undefined
    formData.append("is_html", messageData.is_html ?? false ? "true" : "false");

    if (messageData.thread_id) {
      formData.append("thread_id", messageData.thread_id);
    }

    if (messageData.parent_id) {
      formData.append("parent_id", messageData.parent_id.toString());
    }

    // Add file attachments
    if (messageData.attachments) {
      messageData.attachments.forEach((file) => {
        formData.append("attachments", file);
      });
    }

    // Create headers without Content-Type (let browser set it for multipart)
    const headers: Record<string, string> = {};
    if (this.token) {
      headers["Authorization"] = `Bearer ${this.token}`;
    }

    const response = await fetch(`${this.baseUrl}/api/send`, {
      method: "POST",
      headers,
      body: formData,
    });

    if (!response.ok) {
      if (response.status === 401) {
        this.clearToken();
        throw new Error("Authentication expired. Please login again.");
      }

      // Read response body once and try to parse as JSON
      const responseText = await response.text();
      try {
        const errorData = JSON.parse(responseText);
        if (errorData.message) {
          throw new Error(`Failed to send message: ${errorData.message}`);
        } else if (errorData.error) {
          throw new Error(`Failed to send message: ${errorData.error}`);
        } else {
          throw new Error(
            `Failed to send message: ${JSON.stringify(errorData)}`
          );
        }
      } catch (jsonError) {
        // If JSON parsing fails, use the raw text
        throw new Error(`Failed to send message: ${responseText}`);
      }
    }

    return await response.json();
  }

  // Get messages in a thread
  async getThread(threadId: string): Promise<Message[]> {
    if (!this.token) {
      throw new Error("Not authenticated. Please login first.");
    }

    const response = await fetch(`${this.baseUrl}/api/threads/${threadId}`, {
      headers: this.getAuthHeaders(),
    });

    if (!response.ok) {
      if (response.status === 401) {
        this.clearToken();
        throw new Error("Authentication expired. Please login again.");
      }
      throw new Error(`Failed to fetch thread: ${response.status}`);
    }

    return await response.json();
  }

  // Get attachment URL for download
  getAttachmentUrl(attachmentId: number): string {
    return `${this.baseUrl}/api/attachments/${attachmentId}`;
  }

  // Download attachment
  async downloadAttachment(attachmentId: number): Promise<Blob> {
    if (!this.token) {
      throw new Error("Not authenticated. Please login first.");
    }

    const response = await fetch(
      `${this.baseUrl}/api/attachments/${attachmentId}`,
      {
        headers: {
          Authorization: `Bearer ${this.token}`,
        },
      }
    );

    if (!response.ok) {
      if (response.status === 401) {
        this.clearToken();
        throw new Error("Authentication expired. Please login again.");
      }
      throw new Error(`Failed to download attachment: ${response.status}`);
    }

    return await response.blob();
  }

  // Set up Server-Sent Events for real-time inbox updates
  subscribeToInboxUpdates(
    onMessage: (message: Message) => void,
    onUnreadCount: (count: number) => void
  ): void {
    if (!this.token) {
      throw new Error("Not authenticated. Please login first.");
    }

    this.closeEventSource();

    // Create EventSource with authorization header (Note: EventSource doesn't support custom headers directly)
    // We'll need to pass the token as a query parameter for SSE
    const url = `${this.baseUrl}/api/sse/inbox?token=${encodeURIComponent(
      this.token
    )}`;

    try {
      this.eventSource = new EventSource(url);

      this.eventSource.addEventListener("open", () => {
        console.log("SSE connection opened successfully");
      });

      this.eventSource.addEventListener("connected", (event) => {
        console.log("SSE connected:", event.data);
      });

      this.eventSource.addEventListener("new-message", (event) => {
        try {
          const message = JSON.parse(event.data);
          onMessage(message);
        } catch (error) {
          console.error("Failed to parse new message event:", error);
        }
      });

      this.eventSource.addEventListener("unread-count", (event) => {
        try {
          const data = JSON.parse(event.data);
          onUnreadCount(data.count);
        } catch (error) {
          console.error("Failed to parse unread count event:", error);
        }
      });

      this.eventSource.addEventListener("error", (event) => {
        console.error("SSE connection error:", {
          readyState: this.eventSource?.readyState,
          url: url,
          event: event,
        });

        // Check if it's a connection error vs authentication error
        if (this.eventSource?.readyState === EventSource.CLOSED) {
          console.log("SSE connection was closed by server");
        } else if (this.eventSource?.readyState === EventSource.CONNECTING) {
          console.log("SSE attempting to reconnect...");
        }

        // Attempt to reconnect after a delay
        setTimeout(() => {
          if (
            this.token &&
            this.eventSource?.readyState === EventSource.CLOSED
          ) {
            console.log("Attempting SSE reconnection...");
            this.subscribeToInboxUpdates(onMessage, onUnreadCount);
          }
        }, 5000);
      });
    } catch (error) {
      console.error("Failed to create SSE connection:", error);
    }
  }

  // Close Server-Sent Events connection
  closeEventSource(): void {
    if (this.eventSource) {
      this.eventSource.close();
      this.eventSource = null;
    }
  }

  // Health check endpoint
  async healthCheck(): Promise<{
    status: string;
    timestamp: string;
    version: string;
  }> {
    const response = await fetch(`${this.baseUrl}/api/health`);
    if (!response.ok) {
      throw new Error(`Health check failed: ${response.status}`);
    }
    return await response.json();
  }

  // Helper method to format timestamp
  static formatTimestamp(date: Date): string {
    return date.toISOString();
  }

  // Helper method to parse email address
  static parseEmailAddress(email: string): { username: string; host: string } {
    const parts = email.split("@");
    if (parts.length !== 2) {
      throw new Error("Invalid email format");
    }
    return { username: parts[0], host: parts[1] };
  }
}
