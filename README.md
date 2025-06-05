# 📬 YourMail - Decentralized Mail System

A modern, decentralized email system with real-time messaging, built with Go and Next.js. YourMail features both a custom TCP protocol and modern HTTP REST API, allowing for flexible client implementations while maintaining backward compatibility.

## ✨ Features

### 🔐 **Authentication & Security**

- **JWT Authentication**: Secure token-based authentication system
- **User Registration**: Create new accounts with email and password
- **Session Persistence**: Stay logged in across browser sessions
- **Password Hashing**: Secure bcrypt password storage

### 📨 **Real-Time Messaging**

- **Server-Sent Events (SSE)**: Live inbox updates without page refresh
- **Instant Notifications**: Desktop notifications for new messages
- **Live Unread Counter**: Real-time unread message count updates
- **Message Read Status**: Track read/unread messages automatically

### 🌐 **Dual Protocol Support**

- **HTTP REST API**: Modern JSON API with JWT authentication
- **Custom TCP Protocol**: Original protocol for direct server communication
- **Cross-Protocol Compatibility**: Both protocols share the same database
- **Federation Ready**: External message support via federation relay

### 🎨 **Modern Frontend**

- **Mobile-First Design**: Responsive UI built with Shadcn/ui components
- **Real-Time UI**: Live message updates and connection status indicators
- **Demo Users**: Quick login with pre-configured test accounts
- **Dark/Light Mode**: Automatic theme switching support

### 💾 **Persistent Storage**

- **SQLite Database**: Reliable local storage with migrations
- **Message Relationships**: Proper foreign keys linking users and messages
- **User Management**: Complete CRUD operations for users and messages
- **Data Integrity**: ACID transactions and referential integrity

## 🚀 Quick Start

### Prerequisites

- **Go 1.21+** - [Download Go](https://golang.org/dl/)
- **Node.js 18+** - [Download Node.js](https://nodejs.org/)
- **npm/yarn** - Package manager for Node.js

### 1. Clone the Repository

```bash
git clone https://github.com/yourusername/yourmail.git
cd yourmail
```

### 2. Build and Start the Backend

```bash
# Install Go dependencies
go mod tidy

# Build the server
go build -o yourmail cmd/server/main.go

# Start the server
./yourmail
```

The server will start on:

- **TCP Protocol**: `localhost:7777`
- **HTTP API**: `http://localhost:8080`
- **Database**: `./data/yourmail.db`

### 3. Start the Frontend

```bash
# Navigate to frontend directory
cd frontend

# Install dependencies
npm install

# Start development server
npm run dev
```

The frontend will be available at: `http://localhost:3000`

### 4. Test the System

Open your browser and go to `http://localhost:3000`. You can:

- **Create a new account** or use demo users:
  - **Alice**: `alice` / `password123`
  - **Bob**: `bob` / `password456`
- **Send messages** between users
- **Experience real-time updates** without page refresh
- **Test both protocols** (HTTP and TCP)

## 📖 API Documentation

### Authentication

#### Register

```bash
POST /api/register
Content-Type: application/json

{
  "username": "newuser",
  "email": "user@example.com",
  "password": "securepassword"
}
```

#### Login

```bash
POST /api/login
Content-Type: application/json

{
  "username": "alice",
  "password": "password123"
}
```

### Messages

#### Get Inbox

```bash
GET /api/messages
Authorization: Bearer <jwt_token>
```

#### Send Message

```bash
POST /api/send
Authorization: Bearer <jwt_token>
Content-Type: application/json

{
  "to": "bob@localhost",
  "subject": "Hello!",
  "body": "This is a test message."
}
```

#### Get Unread Count

```bash
GET /api/messages/unread-count
Authorization: Bearer <jwt_token>
```

#### Mark Message as Read

```bash
POST /api/messages/{id}/read
Authorization: Bearer <jwt_token>
```

### Real-Time Updates

#### Server-Sent Events

```bash
GET /api/sse/inbox?token=<jwt_token>
```

Events received:

- `new-message`: When a new message arrives
- `unread-count`: When unread count changes
- `connected`: Connection confirmation

## 🔧 TCP Protocol

The custom TCP protocol supports the following commands:

```
CONNECT <username> <password>    # Authenticate
SEND <recipient@host>            # Set recipient
SUBJECT <subject_text>           # Set subject
BODY <message_body>              # Set message body
LIST                            # List inbox messages
READ <message_id>               # Read specific message
QUIT                            # Close connection
```

### Example TCP Session

```bash
# Connect via telnet or custom client
telnet localhost 7777

# Commands
CONNECT alice password123
SEND bob@localhost
SUBJECT Hello from TCP
BODY This message was sent via TCP protocol!
LIST
QUIT
```

## 🏗️ Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Next.js       │    │   Go Backend    │    │   SQLite DB     │
│   Frontend      │◄──►│                 │◄──►│                 │
│                 │    │  ┌───────────┐  │    │  ┌───────────┐  │
│  - Auth Forms   │    │  │HTTP Server│  │    │  │   Users   │  │
│  - Real-time UI │    │  │    :8080  │  │    │  │ Messages  │  │
│  - SSE Client   │    │  └───────────┘  │    │  └───────────┘  │
│  - Message UI   │    │  ┌───────────┐  │    │                 │
└─────────────────┘    │  │TCP Server │  │    └─────────────────┘
                       │  │   :7777   │  │
┌─────────────────┐    │  └───────────┘  │
│   TCP Clients   │◄──►│                 │
│                 │    │  ┌───────────┐  │
│  - Telnet       │    │  │    SSE    │  │
│  - Custom Apps  │    │  │  Events   │  │
│  - Scripts      │    │  └───────────┘  │
└─────────────────┘    └─────────────────┘
```

## 📁 Project Structure

```
yourmail/
├── cmd/
│   └── server/
│       └── main.go              # Main server entry point
├── config/
│   └── config.go                # Configuration management
├── internal/
│   ├── auth/                    # JWT authentication
│   ├── database/                # Database models & repositories
│   ├── federation/              # Federation/relay system
│   ├── httpapi/                 # HTTP API server
│   └── protocol/                # TCP protocol server
├── frontend/
│   ├── src/
│   │   ├── components/          # React components
│   │   ├── lib/                 # API client & utilities
│   │   └── types/               # TypeScript types
│   ├── package.json
│   └── next.config.ts
├── go.mod
├── go.sum
└── README.md
```

## 🔧 Configuration

The server can be configured via environment variables:

```bash
# Server configuration
TCP_PORT=7777                    # TCP protocol port
HTTP_PORT=8080                   # HTTP API port
SERVER_HOST=localhost            # Server hostname

# Database
DATABASE_PATH=./data/yourmail.db # SQLite database path

# Authentication
JWT_SECRET=your-secret-key       # JWT signing secret
JWT_EXPIRATION=24h               # Token expiration time

# Environment
ENVIRONMENT=development          # development/production
```

## 🧪 Testing

### Run Backend Tests

```bash
go test ./...
```

### Run Frontend Tests

```bash
cd frontend
npm test
```

### Manual Testing

Use the included Python test scripts:

```bash
# Test TCP protocol
python3 tcp_test.py

# Test full system
python3 test_client.py
```

## 🔄 Development

### Backend Development

```bash
# Install development dependencies
go mod tidy

# Run with live reload (install air)
go install github.com/cosmtrek/air@latest
air

# Run tests
go test ./... -v

# Build for production
go build -ldflags="-s -w" -o yourmail cmd/server/main.go
```

### Frontend Development

```bash
cd frontend

# Development server
npm run dev

# Build for production
npm run build

# Start production server
npm start

# Type checking
npm run type-check

# Linting
npm run lint
```

## 🚀 Deployment

### Docker (Coming Soon)

```bash
# Build and run with Docker Compose
docker-compose up --build
```

### Manual Deployment

1. **Build the backend**:

   ```bash
   go build -ldflags="-s -w" -o yourmail cmd/server/main.go
   ```

2. **Build the frontend**:

   ```bash
   cd frontend && npm run build
   ```

3. **Deploy**: Copy binaries and static files to your server

4. **Configure**: Set environment variables for production

5. **Run**: Start the server with process manager (systemd, pm2, etc.)

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request. For major changes, please open an issue first to discuss what you would like to change.

### Development Guidelines

1. **Code Style**: Follow Go and TypeScript best practices
2. **Testing**: Add tests for new features
3. **Documentation**: Update README and code comments
4. **Commits**: Use conventional commit messages
5. **Issues**: Use the issue templates provided

## 📝 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- Built with [Go](https://golang.org/) and [Next.js](https://nextjs.org/)
- UI components from [Shadcn/ui](https://ui.shadcn.com/)
- Icons from [Lucide](https://lucide.dev/)
- Database: [SQLite](https://sqlite.org/)

## 📞 Support

If you have any questions or need help, please:

1. Check the [Issues](https://github.com/yourusername/yourmail/issues) page
2. Create a new issue with a detailed description
3. Join our community discussions

---

**Made with ❤️ by the YourMail community**
