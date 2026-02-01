# API Migration Guide

## Overview

Zorkagent's API has been updated to be compatible with the Opencode SDK specification. This is a **breaking change** that affects the message creation endpoint.

## Breaking Changes

### Endpoint Changes

| Old Endpoint | New Endpoint | Operation |
|-------------|--------------|-----------|
| `POST /session/{session_id}/message` | `POST /session/{sessionID}/prompt` | Create/Send message |
| `GET /session/{session_id}/message` | `GET /session/{sessionID}/message` | List messages (unchanged) |
| `GET /message/{id}` | `GET /message/{id}` | Get message (unchanged) |

### Request Format Changes

#### Old Format
```json
{
  "prompt": "Hello, how are you?",
  "stream": false
}
```

#### New Format (Opencode Compatible)
```json
{
  "messageID": "msg123",
  "model": {
    "providerID": "anthropic",
    "modelID": "claude-sonnet-4"
  },
  "agent": "coding",
  "noReply": false,
  "parts": [
    {
      "text": "Hello, how are you?"
    }
  ]
}
```

### Response Format Changes

#### Old Format
```json
{
  "message": {
    "id": "msg456",
    "session_id": "session123",
    "role": "assistant",
    "content": "I'm doing well!",
    "parts": [...],
    "model": "claude-sonnet-4",
    "provider": "anthropic",
    "created_at": 1234567890,
    "updated_at": 1234567900,
    "finished_at": 1234567900
  },
  "session": {
    "id": "session123",
    ...
  }
}
```

#### New Format (Opencode Compatible)
```json
{
  "info": {
    "id": "msg456",
    "sessionID": "session123",
    "role": "assistant",
    "time": {
      "created": 1234567890,
      "completed": 1234567900
    },
    "modelID": "claude-sonnet-4",
    "providerID": "anthropic",
    "agent": "coding",
    "finish": "end_turn"
  },
  "parts": [
    {
      "type": "text",
      "text": "I'm doing well!"
    }
  ]
}
```

## Migration Steps

### For API Users

1. **Update the endpoint URL**
   ```javascript
   // Before
   const response = await fetch(`/session/${sessionId}/message?directory=${dir}`, {
     method: 'POST',
     body: JSON.stringify({ prompt: "Hello", stream: false })
   });

   // After
   const response = await fetch(`/session/${sessionID}/prompt?directory=${dir}`, {
     method: 'POST',
     body: JSON.stringify({
       parts: [{ text: "Hello" }]
     })
   });
   ```

2. **Update request body structure**
   - Wrap prompt text in a `parts` array
   - Use `parts: [{ text: "your prompt" }]` instead of `prompt: "your prompt"`
   - Remove `stream` parameter (streaming will be handled differently in future)

3. **Update response handling**
   ```javascript
   // Before
   const { message, session } = response;
   console.log(message.content);

   // After
   const { info, parts } = response;
   const textPart = parts.find(p => p.type === 'text');
   console.log(textPart?.text || '');
   ```

4. **Update parameter names**
   - `session_id` → `sessionID` (in URL path)
   - `session.id` → `info.sessionID` (in response)
   - `created_at` → `time.created`
   - `finished_at` → `time.completed`

### New Features

The new API supports additional features:

#### Model Specification
```json
{
  "model": {
    "providerID": "anthropic",
    "modelID": "claude-sonnet-4-20250514"
  },
  "parts": [...]
}
```

#### Agent Selection
```json
{
  "agent": "coding",
  "parts": [...]
}
```

#### NoReply Mode
```json
{
  "noReply": true,
  "parts": [...]
}
```

When `noReply` is true, only the user message is created without running AI inference.

#### File Attachments
```json
{
  "parts": [
    {
      "text": "Please analyze this file"
    },
    {
      "name": "example.txt",
      "data": "SGVsbG8gV29ybGQ="  // base64 encoded
    }
  ]
}
```

## Part Types

The new API uses typed parts instead of untyped maps:

### Text Part
```json
{
  "type": "text",
  "text": "Hello, world!"
}
```

### Reasoning Part
```json
{
  "type": "reasoning",
  "text": "Let me think about this...",
  "time": {
    "created": 1234567890,
    "completed": 1234567895
  }
}
```

### File Part
```json
{
  "type": "file",
  "name": "document.pdf",
  "data": "base64_encoded_data",
  "mimeType": "application/pdf"
}
```

### Tool Part
```json
{
  "type": "tool",
  "name": "read_file",
  "input": "{\"path\": \"/tmp/file.txt\"}"
}
```

### Tool Result Part
```json
{
  "type": "tool_result",
  "name": "read_file",
  "output": "File content here"
}
```

## Examples

### Basic Message
```bash
curl -X POST "http://localhost:8080/session/session123/prompt?directory=/project" \
  -H "Content-Type: application/json" \
  -d '{
    "parts": [
      { "text": "Hello, how are you?" }
    ]
  }'
```

### With Model Specification
```bash
curl -X POST "http://localhost:8080/session/session123/prompt?directory=/project" \
  -H "Content-Type: application/json" \
  -d '{
    "model": {
      "providerID": "anthropic",
      "modelID": "claude-sonnet-4"
    },
    "parts": [
      { "text": "What is 2+2?" }
    ]
  }'
```

### NoReply Mode
```bash
curl -X POST "http://localhost:8080/session/session123/prompt?directory=/project" \
  -H "Content-Type: application/json" \
  -d '{
    "noReply": true,
    "parts": [
      { "text": "This creates a user message only" }
    ]
  }'
```

## Testing

Use the provided test script to verify the API changes:

```bash
./docs/test_prompt_api.sh
```

## Rollback

If you need to rollback to the old API, revert the changes to:
- `api/handlers/messages.go`
- `api/server.go`
- `docs/openapi3.json`

And remove:
- `api/models/opencode_types.go`
- `api/models/conversion.go`

## Support

If you encounter issues during migration:
1. Check the OpenAPI specification at `/swagger` or `/redoc`
2. Review the error messages carefully
3. Ensure your request body matches the new format
4. Verify parameter names (especially `sessionID` vs `session_id`)

## Compatibility Notes

- **Breaking Change**: This is not backward compatible
- **Client Updates Required**: All clients using the old API must be updated
- **Testing**: Test thoroughly before deploying to production
- **SDK Generation**: The OpenAPI specification has been updated for SDK generation
