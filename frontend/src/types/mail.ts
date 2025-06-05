export interface Message {
  id: number;
  from_user_id?: number;
  to_user_id?: number;
  from: string;
  to: string;
  subject: string;
  body: string;
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
}

export interface ServerConfig {
  httpHost: string;
  httpPort: number;
  tcpHost: string;
  tcpPort: number;
}
