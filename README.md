# chirpy



## 📝 Description

Chirpy is a robust backend service engineered in Go, providing a solid foundation for building scalable applications. It incorporates essential features such as database integration for persistent data storage and a comprehensive authentication system to manage user access and security. Chirpy is designed for developers seeking a reliable and efficient backend solution, offering a streamlined approach to handling user management and data persistence in their projects.

## ✨ Features

- 🗄️ Database
- 🔐 Auth


## 🛠️ Tech Stack

- 🐹 Go


## 📦 Key Dependencies

```
(: latest
```

## 🚀 Run Commands

- **Run**: `go run .`
- **Build**: `go build`


## 📁 Project Structure

```
.
├── assets
│   └── logo.png
├── chirpy
├── go.mod
├── go.sum
├── internal
│   ├── auth
│   │   ├── auth.go
│   │   └── auth_test.go
│   └── database
│       ├── chirps.sql.go
│       ├── db.go
│       ├── models.go
│       ├── refresh_token.sql.go
│       └── users.sql.go
├── main.go
├── sql
│   ├── queries
│   │   ├── chirps.sql
│   │   ├── refresh_token.sql
│   │   └── users.sql
│   └── schema
│       ├── 001_users.sql
│       ├── 002_chirps.sql
│       ├── 003_passwords.sql
│       ├── 004_refresh_tokens.sql
│       └── 005_is_chirpy_red.sql
├── sqlc.yaml
└── static
    └── index.html
```

## 🛠️ Development Setup

### Go Setup
1. Install Go (v1.18+ recommended)
2. Install dependencies: `go mod download`
3. Run the project: `go run .`


## 👥 Contributing

Contributions are welcome! Here's how you can help:

1. **Fork** the repository
2. **Clone** your fork: `git clone https://github.com/kanjelkheir/chirpy.git`
3. **Create** a new branch: `git checkout -b feature/your-feature`
4. **Commit** your changes: `git commit -am 'Add some feature'`
5. **Push** to your branch: `git push origin feature/your-feature`
6. **Open** a pull request

Please ensure your code follows the project's style guidelines and includes tests where applicable.
