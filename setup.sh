#!/bin/bash

# YourMail Setup Script
# This script helps you quickly set up and run the YourMail system

set -e

echo "📬 YourMail Setup Script"
echo "========================"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Check prerequisites
check_prerequisites() {
    print_status "Checking prerequisites..."
    
    if ! command_exists go; then
        print_error "Go is not installed. Please install Go 1.21+ from https://golang.org/"
        exit 1
    fi
    
    GO_VERSION=$(go version | cut -d' ' -f3 | sed 's/go//')
    print_success "Go $GO_VERSION found"
    
    if ! command_exists node; then
        print_error "Node.js is not installed. Please install Node.js 18+ from https://nodejs.org/"
        exit 1
    fi
    
    NODE_VERSION=$(node -v)
    print_success "Node.js $NODE_VERSION found"
    
    if ! command_exists npm; then
        print_error "npm is not installed. Please install npm or yarn"
        exit 1
    fi
    
    NPM_VERSION=$(npm -v)
    print_success "npm $NPM_VERSION found"
}

# Setup backend
setup_backend() {
    print_status "Setting up Go backend..."
    
    # Install Go dependencies
    go mod tidy
    print_success "Go dependencies installed"
    
    # Build the server
    go build -o yourmail cmd/server/main.go
    print_success "Server binary built successfully"
}

# Setup frontend
setup_frontend() {
    print_status "Setting up Next.js frontend..."
    
    cd frontend
    
    # Install Node dependencies
    npm install
    print_success "Node.js dependencies installed"
    
    # Build the frontend
    npm run build
    print_success "Frontend built successfully"
    
    cd ..
}

# Create data directory
setup_data() {
    print_status "Setting up data directory..."
    
    if [ ! -d "data" ]; then
        mkdir -p data
        print_success "Data directory created"
    else
        print_warning "Data directory already exists"
    fi
}

# Create environment file
setup_env() {
    print_status "Setting up environment configuration..."
    
    if [ ! -f ".env" ]; then
        cat > .env << EOF
# YourMail Configuration
TCP_PORT=7777
HTTP_PORT=8080
SERVER_HOST=localhost
DATABASE_PATH=./data/yourmail.db
JWT_SECRET=$(openssl rand -base64 32 2>/dev/null || echo "your-random-secret-key-change-in-production")
JWT_EXPIRATION=24h
ENVIRONMENT=development
EOF
        print_success "Environment file created (.env)"
    else
        print_warning "Environment file already exists (.env)"
    fi
}

# Start the system
start_system() {
    print_status "Starting YourMail system..."
    
    # Start backend in background
    print_status "Starting backend server..."
    nohup ./yourmail > backend.log 2>&1 &
    BACKEND_PID=$!
    echo $BACKEND_PID > backend.pid
    
    # Wait a moment for backend to start
    sleep 3
    
    # Check if backend is running
    if kill -0 $BACKEND_PID 2>/dev/null; then
        print_success "Backend server started (PID: $BACKEND_PID)"
        print_status "Backend logs: tail -f backend.log"
    else
        print_error "Failed to start backend server"
        exit 1
    fi
    
    # Start frontend
    print_status "Starting frontend..."
    cd frontend
    nohup npm run dev > ../frontend.log 2>&1 &
    FRONTEND_PID=$!
    echo $FRONTEND_PID > ../frontend.pid
    cd ..
    
    # Wait a moment for frontend to start
    sleep 5
    
    # Check if frontend is running
    if kill -0 $FRONTEND_PID 2>/dev/null; then
        print_success "Frontend server started (PID: $FRONTEND_PID)"
        print_status "Frontend logs: tail -f frontend.log"
    else
        print_error "Failed to start frontend server"
        exit 1
    fi
}

# Stop the system
stop_system() {
    print_status "Stopping YourMail system..."
    
    if [ -f "backend.pid" ]; then
        BACKEND_PID=$(cat backend.pid)
        if kill -0 $BACKEND_PID 2>/dev/null; then
            kill $BACKEND_PID
            print_success "Backend server stopped"
        fi
        rm -f backend.pid
    fi
    
    if [ -f "frontend.pid" ]; then
        FRONTEND_PID=$(cat frontend.pid)
        if kill -0 $FRONTEND_PID 2>/dev/null; then
            kill $FRONTEND_PID
            print_success "Frontend server stopped"
        fi
        rm -f frontend.pid
    fi
    
    # Kill any remaining processes
    pkill -f "yourmail" 2>/dev/null || true
    pkill -f "next dev" 2>/dev/null || true
    
    print_success "YourMail system stopped"
}

# Show status
show_status() {
    print_status "YourMail System Status"
    echo "======================"
    
    # Check backend
    if [ -f "backend.pid" ]; then
        BACKEND_PID=$(cat backend.pid)
        if kill -0 $BACKEND_PID 2>/dev/null; then
            print_success "Backend: Running (PID: $BACKEND_PID)"
            echo "  - TCP Protocol: localhost:7777"
            echo "  - HTTP API: http://localhost:8080"
        else
            print_error "Backend: Not running (stale PID file)"
        fi
    else
        print_warning "Backend: Not running"
    fi
    
    # Check frontend
    if [ -f "frontend.pid" ]; then
        FRONTEND_PID=$(cat frontend.pid)
        if kill -0 $FRONTEND_PID 2>/dev/null; then
            print_success "Frontend: Running (PID: $FRONTEND_PID)"
            echo "  - Web UI: http://localhost:3000"
        else
            print_error "Frontend: Not running (stale PID file)"
        fi
    else
        print_warning "Frontend: Not running"
    fi
    
    echo ""
    echo "Demo accounts:"
    echo "  - alice / password123"
    echo "  - bob / password456"
}

# Show help
show_help() {
    echo "YourMail Setup Script"
    echo ""
    echo "Usage: $0 [command]"
    echo ""
    echo "Commands:"
    echo "  setup     - Full setup (install dependencies, build, configure)"
    echo "  start     - Start the YourMail system"
    echo "  stop      - Stop the YourMail system" 
    echo "  restart   - Restart the YourMail system"
    echo "  status    - Show system status"
    echo "  logs      - Show logs"
    echo "  clean     - Clean build artifacts"
    echo "  help      - Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0 setup     # First time setup"
    echo "  $0 start     # Start the system"
    echo "  $0 status    # Check if running"
}

# Show logs
show_logs() {
    echo "=== Backend Logs ==="
    if [ -f "backend.log" ]; then
        tail -n 20 backend.log
    else
        echo "No backend logs found"
    fi
    
    echo ""
    echo "=== Frontend Logs ==="
    if [ -f "frontend.log" ]; then
        tail -n 20 frontend.log
    else
        echo "No frontend logs found"
    fi
}

# Clean build artifacts
clean_system() {
    print_status "Cleaning build artifacts..."
    
    # Stop system first
    stop_system
    
    # Remove build artifacts
    rm -f yourmail backend.log frontend.log backend.pid frontend.pid
    rm -rf frontend/.next frontend/node_modules
    
    print_success "Build artifacts cleaned"
}

# Main script logic
case "${1:-setup}" in
    "setup")
        check_prerequisites
        setup_backend
        setup_frontend
        setup_data
        setup_env
        print_success "Setup complete!"
        echo ""
        echo "Next steps:"
        echo "  ./setup.sh start    # Start the system"
        echo "  ./setup.sh status   # Check status"
        echo ""
        echo "Then visit: http://localhost:3000"
        ;;
    "start")
        if [ ! -f "yourmail" ]; then
            print_error "Backend not built. Run './setup.sh setup' first"
            exit 1
        fi
        start_system
        echo ""
        print_success "YourMail is running!"
        echo "  - Frontend: http://localhost:3000"
        echo "  - Backend API: http://localhost:8080"
        echo "  - TCP Protocol: localhost:7777"
        echo ""
        echo "Demo accounts: alice/password123, bob/password456"
        ;;
    "stop")
        stop_system
        ;;
    "restart")
        stop_system
        sleep 2
        start_system
        ;;
    "status")
        show_status
        ;;
    "logs")
        show_logs
        ;;
    "clean")
        clean_system
        ;;
    "help"|"-h"|"--help")
        show_help
        ;;
    *)
        print_error "Unknown command: $1"
        show_help
        exit 1
        ;;
esac 