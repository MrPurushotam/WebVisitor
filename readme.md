# Web Visitor - Website Uptime & Performance Monitoring


Web Visitor is a professional-grade website monitoring tool designed to track the uptime, performance, and availability of websites in real-time. Built with a Go backend, it provides detailed analytics, historical logs, and reliable monitoring services to ensure your web applications stay online and perform optimally.

## ✨ Features

- **🔐 User Management**: Secure registration, authentication, and account management
- **🔍 Website Monitoring**: Track multiple URLs with customizable check intervals (6h/12h)
- **📊 Real-time Status Dashboard**: Instant view of website status (online/offline/error)
- **⚡ Performance Metrics**: Detailed response time tracking and HTTP status code logging
- **📝 Historical Logs**: Comprehensive historical data for all website checks
- **🔔 Status Alerts**: (Coming soon) Email notifications when websites go down
- **📱 RESTful API**: Complete API for integration with other systems
- **📚 API Documentation**: Interactive Swagger documentation

## 🛠️ Technology Stack

- **Backend**: Go 1.23+ with Gin web framework
- **Database**: MySQL for reliable data storage
- **Authentication**: Secure token-based session management
- **Scheduling**: Gocron for reliable periodic website checks
- **Documentation**: OpenAPI 3.0/Swagger

## 🚀 Setup Instructions

### Prerequisites

- Go (version 1.18 or higher)
- MySQL server 5.7+ or MariaDB 10.3+
- Git

### Step 1: Clone the Repository

```bash
git clone https://github.com/MrPurushotam/web-visitor.git
cd web-visitor
```

### Step 2: Install Dependencies

```bash
cd backend
go mod download
```

### Step 3: Configure Environment

Create a `.env.local` file in the `backend` directory:

```bash
cp backend/.env.example backend/.env.local
```

Edit the `.env.local` file with your MySQL connection details:

```
MYSQL_URI="user:password@tcp(localhost:3306)/web_visitor?parseTime=true"
GIN_MODE="debug"  # Use "release" in production
CORNJOB_PASSWORD="YourSecretPassword"
```

### Step 4: Create Database

Log into your MySQL server and create a database:

```sql
CREATE DATABASE web_visitor;
```

### Step 5: Run the Application

```bash
cd backend
go run main.go
```

The server will start on port 8080 by default. You can specify a different port with the `PORT` environment variable.

### Step 6: Access API Documentation

Open your browser and navigate to:
```
http://localhost:8080/docs
```

## 📡 API Endpoints

### Health Check
- `GET /` - Check if API is running

### User Management
- `POST /api/v1/user/create/` - Register new user
- `POST /api/v1/user/login/` - User login
- `GET /api/v1/user/` - Get authenticated user details
- `POST /api/v1/user/logout/` - Logout user
- `POST /api/v1/user/verify/` - Verify user account
- `POST /api/v1/user/resend/{email}` - Resend verification email

### URL Management
- `POST /api/v1/uri/` - Add new URL to monitor
- `GET /api/v1/uri/` - Get all monitored URLs
- `PUT /api/v1/uri/{id}` - Update URL details
- `DELETE /api/v1/uri/{id}` - Delete URL and its logs

### Monitoring Logs
- `GET /api/v1/logs/{id}` - Get monitoring logs for a URL

### Monitoring Service Control
- `GET /disable/{password}` - Pause monitoring service
- `GET /enable/{password}` - Resume monitoring service

## 📁 Project Structure

```
web_visitor/
├── backend/
│   ├── config/         # Database configuration
│   ├── libs/           # Database schema definitions
│   ├── middleware/     # Auth middleware and request handlers
│   ├── routes/         # API route handlers
│   ├── service/        # Background monitoring service
│   ├── utils/          # Helper functions and utilities
│   ├── main.go         # Application entry point
│   ├── swagger.go      # Swagger documentation setup
│   ├── go.mod          # Go module definition
│   └── .env            # Environment configuration
├── swagger.yaml        # API specification and documentation
└── readme.md           # Project documentation
```

## ⏱️ Monitoring Service

The application includes a background service that periodically checks the status of monitored URLs:

- **Intervals**: Configurable intervals (default: 6hr and 12hr)
- **Checks**: HTTP requests with proper headers and timeout handling
- **Metrics**: Response time, status code, and error capture
- **Control**: Enable/disable via API endpoints with password protection

## 🗄️ Database Schema

### Main Tables:
- **users**: User accounts with authentication information
  - Fields: id, name, email, password, verified, tier, created_at, updated_at
- **urls**: Monitored websites and their current status
  - Fields: id, user_id, url, name, interval, status, response_time, last_checked
- **logs**: Historical record of all website checks
  - Fields: id, url_id, status, response_time, response_code, error_message, checked_at
- **auth_tokens**: User sessions and authentication management
  - Fields: id, user_id, token, expires_at, is_active, created_at, last_used_at

## 🔒 Security Features

- Secure password hashing with bcrypt
- Session-based authentication with tokens
- HTTP-only cookies for session management
- URL validation and sanitization
- Protection against private IP monitoring

## ❓ Troubleshooting

### Common Issues:

1. **Database Connection Errors**:
   - Verify MySQL credentials in the `.env.local` file
   - Ensure MySQL server is running
   - Check for proper network connectivity

2. **Permission Issues**:
   - Ensure the user has appropriate permissions on the database

3. **Port Already in Use**:
   - Change the port using the `PORT` environment variable
   - Check if another instance is already running

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the project
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📄 License

This project is licensed under the MIT License - see the LICENSE file for details.

## 📧 Contact

Project maintained by Purushotam Jeswani - [GitHub Profile](https://github.com/MrPurushotam)
