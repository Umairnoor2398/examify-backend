# Examify Backend

RESTful API for Examify — a school exam/paper generation portal. Built with Go, Gin, GORM, and MySQL.

## Tech Stack

- **Go 1.25+**
- **Gin** — HTTP framework
- **GORM** — ORM with MySQL driver
- **JWT** — Authentication (access + refresh tokens)
- **bcrypt** — Password hashing

## Prerequisites

- Go 1.22+
- MySQL 8.0+

## Setup

### 1. Clone and install dependencies

```bash
git clone https://github.com/Umairnoor2398/examify-backend.git
cd examify-backend
go mod download
```

### 2. Configure environment

```bash
cp .env.example .env
# Edit .env with your database credentials and secrets
```

### 3. Create MySQL database

```sql
CREATE DATABASE examify CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
```

### 4. Run the server

```bash
go run cmd/server/main.go
```

The server will start on `http://localhost:8080` and auto-migrate the database schema.

A default admin account is seeded on first run:
- **Email:** `admin@examify.com`
- **Password:** `Admin@123456`

> **Important:** Change the default admin password immediately after first login.

## API Overview

Base URL: `http://localhost:8080/api/v1`

### Authentication
| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/auth/login` | Login with email/password |
| POST | `/auth/refresh` | Refresh access token |
| POST | `/auth/forgot-password` | Request password reset |
| POST | `/auth/reset-password` | Reset password with token |
| GET | `/auth/me` | Get current user info |
| GET | `/auth/permissions` | Get current user permissions |

### Admin
| Method | Endpoint | Description |
|--------|----------|-------------|
| CRUD | `/admin/schools` | Manage school accounts |
| CRUD | `/admin/curricula` | Manage curricula |
| CRUD | `/admin/subjects` | Manage subjects |
| CRUD | `/admin/classes` | Manage classes |
| CRUD | `/admin/books` | Manage books |
| CRUD | `/admin/books/:bookId/chapters` | Manage chapters |
| CRUD | `/admin/chapters/:chapterId/questions` | Manage questions |
| POST | `/admin/books/import` | Bulk import books via CSV |
| POST | `/admin/books/:bookId/chapters/import` | Bulk import chapters via CSV |
| POST | `/admin/chapters/:chapterId/questions/import` | Bulk import questions via CSV |

### School Admin
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET/PUT | `/school/profile` | Manage school profile |
| POST | `/school/profile/logo` | Upload school logo |
| CRUD | `/school/teachers` | Manage teacher accounts |
| PATCH | `/school/teachers/:id/toggle` | Enable/disable teacher |
| CRUD | `/school/books` | Manage custom books |
| GET/POST/DELETE | `/school/papers` | Manage submitted papers |

### Teacher
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/teacher/books` | Get available books |
| GET | `/teacher/books/:bookId/chapters` | Get book chapters |
| GET | `/teacher/questions` | Get filtered questions |
| CRUD | `/teacher/papers` | Manage papers |
| POST | `/teacher/papers/:id/submit` | Submit a draft paper |

## CSV Import Formats

### Books CSV
```
title,description,version
Mathematics Grade 10,Complete math textbook,1.0
Physics Grade 11,Physics for FSc,2.1
```

### Chapters CSV
```
name,description
Chapter 1: Algebra,Introduction to algebra
Chapter 2: Geometry,Basic geometry concepts
```

### Questions CSV
```
type,text,difficulty,answer_or_options,tags
mcq,What is 2+2?,easy,"3|4|5|6|1",math,arithmetic
short,Explain Newton's first law,medium,An object remains at rest...,physics
descriptive,Describe the water cycle,hard,The water cycle involves...,science
```

For MCQ: `answer_or_options` = `opt1|opt2|opt3|opt4|correct_index` (0-based)

## Roles & Permissions

| Role | Capabilities |
|------|-------------|
| Admin | Full access — manage schools, curricula, subjects, classes, books, chapters, questions |
| School Admin | Manage school profile, teachers (up to limit), custom books, view/print/delete submitted papers |
| Teacher | View books/questions, create and manage own papers |

## Security

- JWT access tokens (15 min) + refresh tokens (7 days)
- bcrypt password hashing
- Role-based access control on all protected endpoints
- CORS configured for frontend origin
- SQL injection prevention via GORM parameterized queries
- Password enumeration prevention on forgot-password endpoint
