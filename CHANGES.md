
# PicoClaw Telegram SQLite Persistence

## Overview

This update adds **SQLite conversation persistence** for Telegram users to PicoClaw. It persists user messages in a local SQLite database, loads recent history when building prompts for PicoClaw, and ensures the database is persisted across redeploys in Coolify.

**Key outcomes**
- Save incoming Telegram messages to `history.db`.
- Retrieve last N messages to build context for PicoClaw responses.
- Persist `history.db` using Coolify persistent storage.
- Provide a verification checklist for an AI agent to validate the enhancement.

---

## Files Added and Modified

**New file**
- `internal/db/sqlite/sqlite.go` — SQLite helper package with `InitDB`, `CloseDB`, `SaveMessage`, `GetHistory`.

**Modified files**
- `cmd/picoclaw/main.go` — initialize DB on startup and pass `DB_PATH` from env.
- `internal/handlers/telegram.go` — call `SaveMessage` on incoming messages and save bot replies.
- `internal/prompt/context.go` — call `GetHistory` to assemble prompt.
- `Dockerfile` — builder stage updated to include `sqlite-dev` and runtime stage installs `sqlite` and ensures `/app/history.db` exists.
- `.env` — new environment variables for DB and Telegram settings.

---

## Environment Variables

Place these in your `.env` or Coolify environment variables panel.

```env
TELEGRAM_TOKEN=your_telegram_bot_token_here
TELEGRAM_USE_WEBHOOK=false
WEBHOOK_URL=https://your-coolify-domain.example.com/webhook
DB_PATH=/app/history.db
SQLITE_INIT=true
PORT=18790
LOG_LEVEL=info
MAX_HISTORY=20
FORGET_COMMAND_ENABLED=true
```

**Notes**
- **DB_PATH** must match the mounted persistent volume path inside the container.
- **TELEGRAM_USE_WEBHOOK** controls whether the app uses webhook or polling.

---

## Database Schema

Run automatically by `InitDB`. Schema:

```sql
CREATE TABLE IF NOT EXISTS telegram_history (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id INTEGER NOT NULL,
  message TEXT NOT NULL,
  timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

**Behavior**
- Messages are appended for both user and bot replies.
- `GetHistory` returns messages ordered by `timestamp DESC` and limited by `MAX_HISTORY`.

---

## How It Works

1. **Startup**
   - `main.go` reads `DB_PATH` and calls `sqlite.InitDB(dbPath)`.
2. **Incoming message**
   - Telegram handler receives update.
   - Call `sqlite.SaveMessage(userID, messageText)` to persist the incoming message.
3. **Context building**
   - Handler calls `sqlite.GetHistory(userID, MAX_HISTORY)` to fetch recent messages.
   - `internal/prompt/context.go` formats history and current message into a prompt for PicoClaw.
4. **Response**
   - PicoClaw generates a reply.
   - Handler optionally calls `sqlite.SaveMessage(userID, replyText)` to persist the bot reply.
5. **Shutdown**
   - `sqlite.CloseDB()` is called on graceful shutdown.

---

## Dockerfile Changes Summary

**Builder stage**
- Add `build-base` and `sqlite-dev` to support `github.com/mattn/go-sqlite3` with CGO.

**Runtime stage**
- Install `sqlite` package for CLI tools if needed.
- Ensure `/app` exists and create `/app/history.db` so the file is present for volume mount.

Example builder snippet:

```dockerfile
RUN apk add --no-cache git make build-base sqlite-dev
RUN CGO_ENABLED=1 go build -o build/picoclaw ./cmd/picoclaw
```

Example runtime snippet:

```dockerfile
RUN apk add --no-cache ca-certificates tzdata sqlite
WORKDIR /app
RUN touch /app/history.db
COPY --from=builder /src/build/picoclaw /usr/local/bin/picoclaw
ENTRYPOINT ["picoclaw"]
CMD ["gateway"]
```

---

## Coolify Deployment Steps

1. **Create Application**
   - Add new app in Coolify from your GitHub repo.
   - Select Dockerfile build.

2. **Persistent Storage**
   - Configure persistent storage mapping:
     - **Container path**: `/app/history.db` or mount `/app` directory.
     - **Host path**: choose a stable host path or Coolify volume.

3. **Environment Variables**
   - Add `.env` values in Coolify environment variables panel.

4. **Networking**
   - If using webhooks, expose `PORT` and set Telegram webhook to `WEBHOOK_URL`.
   - If using polling, no external port is required.

5. **Deploy**
   - Deploy and verify logs show DB initialization and gateway startup.

---

## Verification Checklist for AI Agent

Use this checklist to validate the enhancement end to end.

### Setup checks
- **Env variables present**
  - `TELEGRAM_TOKEN`, `DB_PATH`, `MAX_HISTORY` exist in environment.
- **DB file exists**
  - Confirm `/app/history.db` exists in container filesystem and is writable.

### Functional checks
1. **DB initialization**
   - On startup logs show successful DB initialization or table creation.
   - SQL schema exists: `SELECT name FROM sqlite_master WHERE type='table' AND name='telegram_history';`

2. **Save incoming message**
   - Send a test message to the bot.
   - Confirm a new row inserted:
     ```sql
     SELECT id, user_id, message, timestamp FROM telegram_history ORDER BY timestamp DESC LIMIT 5;
     ```
   - Confirm `user_id` and `message` match the test message.

3. **Context retrieval**
   - Trigger a message that causes the bot to build a response.
   - Confirm `GetHistory` returns up to `MAX_HISTORY` messages and that the prompt includes them in correct order.

4. **Save bot reply**
   - Confirm bot replies are also saved as rows in `telegram_history`.

5. **Persistence across redeploy**
   - Redeploy the app.
   - Confirm previously saved messages remain in `history.db`.

6. **Forget command**
   - If `FORGET_COMMAND_ENABLED=true`, send `/forget`.
   - Confirm rows for that `user_id` are deleted:
     ```sql
     SELECT COUNT(*) FROM telegram_history WHERE user_id = <test_user_id>;
     ```

7. **Concurrency and errors**
   - Send multiple messages concurrently and confirm no DB errors in logs.
   - Simulate DB failure by making `history.db` read-only and confirm the app logs errors but continues to operate with empty history fallback.

### Logs to inspect
- DB init success: `InitDB` or `created table telegram_history`.
- Save errors: `failed to save message`.
- GetHistory errors: `failed to get history`.
- Webhook or polling startup messages.

### Acceptance criteria
- Messages are persisted and retrievable.
- DB persists across redeploys.
- Bot continues to respond if DB read fails fallback to empty history.
- `/forget` removes user history when enabled.
- No unhandled panics or fatal DB errors in normal operation.

---

## Testing Commands and Examples

**Check DB inside container**
```sh
docker exec -it <container_id> sh
sqlite3 /app/history.db "SELECT id, user_id, message, timestamp FROM telegram_history ORDER BY timestamp DESC LIMIT 10;"
```

**Sample SQL to verify**
```sql
SELECT COUNT(*) FROM telegram_history;
SELECT message FROM telegram_history WHERE user_id = 123456789 ORDER BY timestamp DESC LIMIT 20;
DELETE FROM telegram_history WHERE user_id = 123456789;
```

**Simulate a message via webhook**
```sh
curl -X POST -H "Content-Type: application/json" -d '{"message":{"from":{"id":123456789},"text":"hello test"}}' https://your-coolify-domain.example.com/webhook
```

---

## Troubleshooting

- **DB file not writable**
  - Ensure Coolify volume is mounted and container user has write permission.
  - Check file ownership and permissions inside container.

- **`github.com/mattn/go-sqlite3` build errors**
  - Ensure `CGO_ENABLED=1` and `sqlite-dev` and `build-base` are installed in builder stage.

- **Missing messages after redeploy**
  - Confirm the persistent volume mapping points to the same host path and container path.
  - Verify `history.db` is not being overwritten by a container startup script.

- **High latency on DB operations**
  - Use prepared statements or batch inserts.
  - Consider moving to Postgres if scaling horizontally.

---

## Security and Privacy

- **Data minimization**
  - Store only `user_id`, `message`, and `timestamp`.
- **User control**
  - Implement `/forget` to delete a user’s history.
- **Access control**
  - Do not expose `history.db` via HTTP.
  - Limit access to Coolify volumes and backups.
- **Backups**
  - Periodically snapshot the persistent volume.

---

## Rollback Plan

1. Redeploy previous image without DB changes.
2. If `history.db` was overwritten, restore from backup.
3. Revert code changes in Git and redeploy.

---

## Notes for the AI Agent Verifier

- Follow the verification checklist step by step and mark each item as **Pass**, **Fail**, or **N/A**.
- Capture logs and SQL query outputs as evidence for each test.
- Report any deviations from acceptance criteria with reproduction steps and suggested fixes.