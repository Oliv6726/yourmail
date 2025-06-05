export interface Attachment {
  id: number;
  message_id: number;
  filename: string;
  original_name: string;
  content_type: string;
  file_size: number;
  created_at: string;
}

export interface Message {
  id: number;
  from_user_id?: number;
  to_user_id?: number;
  from: string;
  to: string;
  subject: string;
  body: string;
  is_html?: boolean;
  thread_id?: string;
  parent_id?: number;
  read: boolean;
  timestamp: string;
  created_at?: string;

  // Optional virtual fields populated by backend
  from_user?: {
    id: number;
    username: string;
    email: string;
  };
  to_user?: {
    id: number;
    username: string;
    email: string;
  };

  // Threading fields
  replies?: Message[];
  attachment_count?: number;
  attachments?: Attachment[];
}

export interface User {
  username: string;
  serverHost: string;
}

export interface ServerConnection {
  isConnected: boolean;
  host: string;
  port: number;
  user?: User;
}

export interface ComposeMessage {
  to: string;
  subject: string;
  body: string;
  is_html?: boolean;
  thread_id?: string;
  parent_id?: number;
}

export interface ServerConfig {
  httpHost: string;
  httpPort: number;
  tcpHost: string;
  tcpPort: number;
}
