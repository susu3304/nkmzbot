# nkmzbot API Documentation

## Overview

The nkmzbot API is a RESTful API for authentication and guild discovery. All endpoints return JSON responses.

## Authentication

All endpoints require authentication via Discord OAuth2. The API uses JWT tokens that are stored in HTTP-only cookies for security.

### Authentication Flow

1. **Get the OAuth2 URL**
```bash
curl http://localhost:3000/api/auth/login
```

Response:
```json
{
  "auth_url": "https://discord.com/api/oauth2/authorize?...",
  "state": "random_state_string"
}
```

2. **Complete OAuth2 flow**
   - Direct the user to the `auth_url`
   - Discord will redirect back to your `DISCORD_REDIRECT_URI` with a code
   - The callback endpoint will set a JWT token in an HTTP-only cookie

3. **Use the authentication**
   - The JWT token is automatically sent via cookie in subsequent requests
   - Alternatively, include the token in the `Authorization` header as `Bearer <token>`

## Endpoints

### Authentication

#### GET /api/auth/login
Get the Discord OAuth2 authorization URL.

**Response:**
```json
{
  "auth_url": "https://discord.com/api/oauth2/authorize?...",
  "state": "random_state"
}
```

#### GET /api/auth/callback
OAuth2 callback endpoint. Called by Discord after user authorization.

**Query Parameters:**
- `code`: Authorization code from Discord

**Response:**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user_id": "123456789",
  "username": "username"
}
```

#### POST /api/auth/logout
Logout endpoint (client should discard the JWT token).

**Response:**
```json
{
  "message": "logged out"
}
```

### Guilds

#### GET /api/user/guilds
Get list of guilds the authenticated user belongs to.

**Headers:**
- `Authorization: Bearer <token>`

**Response:**
```json
[
  {
    "id": "123456789",
    "name": "My Guild",
    "owner": false
  }
]
```

## Error Responses

All error responses follow this format:

```json
{
  "error": "Error message"
}
```

Common HTTP status codes:
- `400 Bad Request` - Invalid request body or parameters
- `401 Unauthorized` - Missing or invalid authentication token
- `403 Forbidden` - User doesn't have access to the requested resource
- `404 Not Found` - Resource not found
- `500 Internal Server Error` - Server error

## CORS

The API supports CORS to allow requests from web browsers. By default, it allows all origins (`*`) for development purposes. For production, you should configure specific allowed origins in the code.

## Rate Limiting

Currently, there is no rate limiting implemented. Consider adding rate limiting for production use.
