# PCF-MCP API Documentation

This document provides comprehensive API documentation for the PCF-MCP server's MCP tools and HTTP endpoints.

## Table of Contents

- [HTTP Endpoints](#http-endpoints)
- [MCP Tools](#mcp-tools)
- [Error Handling](#error-handling)
- [Authentication](#authentication)

## HTTP Endpoints

The HTTP transport exposes the following REST API endpoints:

### Health Check

Check server health status.

**Request:**
```http
GET /health
```

**Response:**
```json
{
  "status": "healthy",
  "timestamp": "2024-01-01T00:00:00Z",
  "version": "0.1.0"
}
```

### Server Info

Get server information and capabilities.

**Request:**
```http
GET /info
```

**Response:**
```json
{
  "name": "pcf-mcp",
  "version": "0.1.0",
  "capabilities": {
    "tools": true,
    "resources": false,
    "prompts": false
  }
}
```

### List Tools

Get available MCP tools.

**Request:**
```http
GET /tools
```

**Response:**
```json
{
  "tools": [
    {
      "name": "list_projects",
      "description": "List all projects in PCF",
      "inputSchema": {
        "type": "object",
        "properties": {},
        "additionalProperties": false
      }
    }
    // ... more tools
  ]
}
```

### Execute Tool

Execute a specific MCP tool.

**Request:**
```http
POST /tools/{tool_name}
Content-Type: application/json

{
  // Tool-specific parameters
}
```

**Response:**
```json
{
  "result": {
    // Tool-specific result
  }
}
```

### Metrics

Prometheus metrics endpoint.

**Request:**
```http
GET /metrics
```

**Response:**
```
# HELP http_requests_total Total number of HTTP requests
# TYPE http_requests_total counter
http_requests_total{method="GET",path="/health",status="200"} 42
```

## MCP Tools

### Project Management

#### list_projects

List all pentest projects in PCF.

**Parameters:**
```json
{}
```

**Response:**
```json
{
  "projects": [
    {
      "id": "proj-123",
      "name": "Example Pentest",
      "description": "Q1 2024 Security Assessment",
      "status": "active",
      "team": ["alice", "bob"],
      "created_at": "2024-01-01T00:00:00Z",
      "updated_at": "2024-01-02T00:00:00Z"
    }
  ],
  "total_count": 1
}
```

#### create_project

Create a new pentest project.

**Parameters:**
```json
{
  "name": "string (required)",
  "description": "string (optional)",
  "team": ["string"] // optional
}
```

**Response:**
```json
{
  "project": {
    "id": "proj-124",
    "name": "New Pentest",
    "description": "Description",
    "status": "active",
    "team": ["alice"],
    "created_at": "2024-01-03T00:00:00Z",
    "updated_at": "2024-01-03T00:00:00Z"
  }
}
```

### Host Management

#### list_hosts

List hosts in a project with optional filters.

**Parameters:**
```json
{
  "project_id": "string (required)",
  "os": "string (optional)",    // Filter by OS
  "service": "string (optional)" // Filter by service
}
```

**Response:**
```json
{
  "hosts": [
    {
      "id": "host-123",
      "project_id": "proj-123",
      "ip": "192.168.1.100",
      "hostname": "web-server",
      "os": "Linux",
      "services": ["ssh", "http", "https"],
      "status": "active"
    }
  ],
  "total_count": 1,
  "project_id": "proj-123"
}
```

#### add_host

Add a new host to a project.

**Parameters:**
```json
{
  "project_id": "string (required)",
  "ip": "string (required)",
  "hostname": "string (optional)",
  "os": "string (optional)",
  "services": ["string"] // optional
}
```

**Response:**
```json
{
  "host": {
    "id": "host-124",
    "project_id": "proj-123",
    "ip": "192.168.1.101",
    "hostname": "db-server",
    "os": "Linux",
    "services": ["ssh", "mysql"],
    "status": "active"
  }
}
```

### Issue Management

#### list_issues

List security issues in a project with optional filters.

**Parameters:**
```json
{
  "project_id": "string (required)",
  "severity": "string (optional)",   // Critical, High, Medium, Low, Info
  "status": "string (optional)",     // Open, Closed, In Progress
  "host_id": "string (optional)"     // Filter by host
}
```

**Response:**
```json
{
  "issues": [
    {
      "id": "issue-123",
      "project_id": "proj-123",
      "host_id": "host-123",
      "title": "SQL Injection",
      "description": "SQL injection vulnerability in login form",
      "severity": "Critical",
      "status": "Open",
      "cve": "CVE-2024-1234",
      "cvss": 9.8,
      "affected_systems": ["web-server"],
      "evidence": {
        "screenshots": ["screenshot1.png"],
        "logs": ["request.log"]
      }
    }
  ],
  "total_count": 1,
  "severity_breakdown": {
    "Critical": 1,
    "High": 0,
    "Medium": 0,
    "Low": 0,
    "Info": 0
  }
}
```

#### create_issue

Create a new security issue.

**Parameters:**
```json
{
  "project_id": "string (required)",
  "title": "string (required)",
  "description": "string (required)",
  "severity": "string (required)",    // Critical, High, Medium, Low, Info
  "host_id": "string (optional)",
  "cve": "string (optional)",
  "cvss": "number (optional)",
  "affected_systems": ["string"],     // optional
  "evidence": {                       // optional
    "screenshots": ["string"],
    "logs": ["string"]
  }
}
```

**Response:**
```json
{
  "issue": {
    "id": "issue-124",
    "project_id": "proj-123",
    "title": "New Security Issue",
    // ... full issue object
  }
}
```

### Credential Management

#### list_credentials

List stored credentials in a project. Values are always redacted.

**Parameters:**
```json
{
  "project_id": "string (required)",
  "type": "string (optional)",      // password, hash, key, token, certificate
  "host_id": "string (optional)",
  "service": "string (optional)"
}
```

**Response:**
```json
{
  "credentials": [
    {
      "id": "cred-123",
      "project_id": "proj-123",
      "host_id": "host-123",
      "type": "password",
      "username": "admin",
      "value": "***REDACTED***",
      "service": "ssh",
      "notes": "Default admin account"
    }
  ],
  "total_count": 1,
  "type_breakdown": {
    "password": 1,
    "hash": 0,
    "key": 0,
    "token": 0,
    "certificate": 0
  }
}
```

#### add_credential

Store a new credential securely.

**Parameters:**
```json
{
  "project_id": "string (required)",
  "type": "string (required)",        // password, hash, key, token, certificate
  "username": "string (required)",
  "value": "string (required)",       // Will be encrypted
  "host_id": "string (optional)",
  "service": "string (optional)",
  "notes": "string (optional)"
}
```

**Response:**
```json
{
  "credential": {
    "id": "cred-124",
    "project_id": "proj-123",
    "type": "password",
    "username": "user",
    "value": "***REDACTED***",
    // ... full credential object
  }
}
```

### Report Generation

#### generate_report

Generate a report for a project.

**Parameters:**
```json
{
  "project_id": "string (required)",
  "format": "string (required)",           // pdf, html, json, markdown
  "include_hosts": "boolean (optional)",   // default: true
  "include_issues": "boolean (optional)",  // default: true
  "include_credentials": "boolean (optional)", // default: false
  "sections": ["string"]                   // optional sections to include
}
```

**Available Sections:**
- `executive_summary`
- `technical_findings`
- `risk_assessment`
- `remediation`
- `appendix`

**Response:**
```json
{
  "report": {
    "id": "report-123",
    "project_id": "proj-123",
    "format": "pdf",
    "status": "completed",
    "url": "https://pcf.example.com/reports/report-123.pdf",
    "created_at": "2024-01-03T00:00:00Z",
    "size": 1048576,
    "sections": ["executive_summary", "technical_findings"]
  }
}
```

## Error Handling

All endpoints return consistent error responses:

```json
{
  "error": "Error message description"
}
```

### HTTP Status Codes

- `200 OK` - Successful request
- `400 Bad Request` - Invalid request parameters
- `401 Unauthorized` - Missing or invalid authentication
- `404 Not Found` - Resource not found
- `500 Internal Server Error` - Server error

### Tool-Specific Errors

Tool execution errors are returned with status 500:

```json
{
  "error": "failed to list projects: connection timeout"
}
```

## Authentication

When authentication is enabled, all endpoints except `/health` and `/metrics` require a Bearer token.

### Configuration

Enable authentication via configuration:

```yaml
server:
  auth_required: true
  auth_token: "your-secret-token"
```

Or via command line:
```bash
./pcf-mcp --server-auth-required true --server-auth-token "your-secret-token"
```

### Usage

Include the token in the Authorization header:

```http
GET /tools
Authorization: Bearer your-secret-token
```

### Error Response

Missing or invalid authentication returns 401:

```json
{
  "error": "Authorization header required"
}
```