# Contributing to YourMail

Thank you for your interest in contributing to YourMail! We welcome contributions from everyone.

## 🚀 Getting Started

### Prerequisites

- Go 1.21 or higher
- Node.js 18 or higher
- Git

### Development Setup

1. **Fork the repository** on GitHub
2. **Clone your fork:**

   ```bash
   git clone https://github.com/yourusername/yourmail.git
   cd yourmail
   ```

3. **Set up the backend:**

   ```bash
   go mod tidy
   go build -o yourmail cmd/server/main.go
   ./yourmail
   ```

4. **Set up the frontend:**
   ```bash
   cd frontend
   npm install
   npm run dev
   ```

## 🔄 Development Workflow

### Making Changes

1. **Create a feature branch:**

   ```bash
   git checkout -b feature/your-feature-name
   ```

2. **Make your changes** following our coding standards

3. **Test your changes:**

   ```bash
   # Backend tests
   go test ./...

   # Frontend tests
   cd frontend && npm test
   ```

4. **Commit your changes:**

   ```bash
   git add .
   git commit -m "feat: add your feature description"
   ```

5. **Push to your fork:**

   ```bash
   git push origin feature/your-feature-name
   ```

6. **Create a Pull Request** on GitHub

### Commit Message Format

We use [Conventional Commits](https://conventionalcommits.org/) format:

```
<type>[optional scope]: <description>

[optional body]

[optional footer(s)]
```

**Types:**

- `feat`: A new feature
- `fix`: A bug fix
- `docs`: Documentation only changes
- `style`: Changes that do not affect the meaning of the code
- `refactor`: A code change that neither fixes a bug nor adds a feature
- `test`: Adding missing tests or correcting existing tests
- `chore`: Changes to the build process or auxiliary tools

**Examples:**

- `feat: add real-time message notifications`
- `fix: resolve SSE connection timeout issues`
- `docs: update API documentation`

## 🎯 Contribution Areas

We welcome contributions in these areas:

### 🔧 Backend (Go)

- Protocol improvements
- Database optimizations
- API enhancements
- Security improvements
- Performance optimizations

### 🎨 Frontend (Next.js/React)

- UI/UX improvements
- Mobile responsiveness
- Accessibility enhancements
- New features
- Performance optimizations

### 📚 Documentation

- API documentation
- Code comments
- Tutorial improvements
- Example applications

### 🧪 Testing

- Unit tests
- Integration tests
- End-to-end tests
- Performance tests

## 📋 Coding Standards

### Go Code Style

- Follow standard Go formatting (`gofmt`)
- Use meaningful variable and function names
- Add comments for exported functions
- Handle errors appropriately
- Write tests for new functionality

```go
// Good example
func (s *Server) GetUserByID(userID int) (*User, error) {
    if userID <= 0 {
        return nil, fmt.Errorf("invalid user ID: %d", userID)
    }

    user, err := s.userRepo.GetByID(userID)
    if err != nil {
        return nil, fmt.Errorf("failed to get user: %w", err)
    }

    return user, nil
}
```

### TypeScript/React Code Style

- Use TypeScript for all new code
- Follow React best practices
- Use proper component composition
- Handle loading and error states
- Write descriptive prop types

```typescript
// Good example
interface MessageProps {
  message: Message;
  onRead: (id: number) => void;
  isSelected?: boolean;
}

export function MessageItem({
  message,
  onRead,
  isSelected = false,
}: MessageProps) {
  const handleClick = useCallback(() => {
    if (!message.read) {
      onRead(message.id);
    }
  }, [message.id, message.read, onRead]);

  return (
    <div
      className={`message-item ${isSelected ? "selected" : ""}`}
      onClick={handleClick}
    >
      {/* Component content */}
    </div>
  );
}
```

## 🐛 Bug Reports

When reporting bugs, please include:

1. **Description**: A clear description of the bug
2. **Steps to reproduce**: Detailed steps to reproduce the issue
3. **Expected behavior**: What you expected to happen
4. **Actual behavior**: What actually happened
5. **Environment**:
   - OS and version
   - Go version
   - Node.js version
   - Browser (if frontend issue)

## 💡 Feature Requests

For feature requests, please:

1. **Check existing issues** to avoid duplicates
2. **Describe the feature** clearly and concisely
3. **Explain the use case** and why it would be beneficial
4. **Provide examples** if applicable

## 🔍 Code Review Process

All submissions require review. We use GitHub pull requests for this purpose:

1. **Automated checks** must pass (tests, linting)
2. **Code review** by at least one maintainer
3. **Testing** on different environments
4. **Documentation** updates if needed

## 📦 Release Process

1. Version bump following [Semantic Versioning](https://semver.org/)
2. Update CHANGELOG.md
3. Create GitHub release with release notes
4. Deploy to staging for testing
5. Deploy to production

## 🏷️ Issue Labels

We use these labels to categorize issues:

- `bug`: Something isn't working
- `enhancement`: New feature or request
- `documentation`: Improvements or additions to documentation
- `good first issue`: Good for newcomers
- `help wanted`: Extra attention is needed
- `question`: Further information is requested
- `wontfix`: This will not be worked on

## 🤝 Community Guidelines

- Be respectful and inclusive
- Help others learn and grow
- Give constructive feedback
- Focus on the issue, not the person
- Follow the [Code of Conduct](CODE_OF_CONDUCT.md)

## 📞 Getting Help

If you need help:

1. Check the [documentation](README.md)
2. Search [existing issues](https://github.com/yourusername/yourmail/issues)
3. Create a new issue with the `question` label
4. Join our community discussions

## 🙏 Recognition

Contributors will be recognized in:

- README.md contributors section
- Release notes
- GitHub contributors page

Thank you for helping make YourMail better! 🚀
