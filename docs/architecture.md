# Lehu Video Backend Architecture

## Overview

Lehu Video keeps the existing Kratos microservice layout:

- `videoApi`: public HTTP API gateway, JWT validation, response encoding, WebSocket entry.
- `base`: account, verification code, file metadata, MinIO presigned upload.
- `videoCore`: video, feed, favorite, collection, comment, follow, counters.
- `videoChat`: friend, group, conversation, message persistence.

The refactor direction is to keep the service split, but make boundaries explicit: API gateway performs protocol adaptation, core services own business rules, repositories own persistence, and shared packages provide cross-service concerns such as errors, JWT and password hashing.

## Core Flows

### Auth

Registration validates a verification code, creates an account in `base`, then creates the user profile in `videoCore`. Login validates credentials in `base`, fetches the user profile in `videoCore`, and issues a JWT from the configured API secret. HTTP and WebSocket token parsing share the same claims implementation.

Passwords are stored with Argon2id for new accounts. Legacy MD5+salt hashes are still accepted and are upgraded after a successful login.

### Video Upload And Publish

The client asks `videoApi` for a presigned upload URL. `base` records file metadata and signs a MinIO URL. After upload completion, `videoApi` confirms the file and asks `videoCore` to create the video record. Cover URL is optional; frontend can render a fallback cover.

### Feed

`videoApi` uses `videoCore.FeedService.GetFeed` as the single feed entrypoint. Feed types are:

- `0`: following
- `1`: recommended
- `2`: hot
- `3`: mixed

Guests requesting following feed are downgraded to recommended feed. If the hot pool is empty, feed falls back to latest videos.

### Counters And Consistency

Video view, favorite, collection and comment counts use Redis counters for hot reads and asynchronous persistence for write pressure. Reconcile jobs periodically repair drift between Redis counters and MySQL source data. Counter failures should delay statistics, not fail core user actions when the database write path has succeeded.

## Engineering Notes

- Public HTTP responses keep `code`, `message`, `data`, `timestamp`, and `request_id`.
- Shared business errors live in `pkg/apperror`; service metadata and HTTP errors are derived from those codes.
- Shared JWT helpers live in `pkg/auth`.
- Shared password hashing helpers live in `pkg/password`.
- Service startup should fail fast if MySQL, Redis, Consul or gRPC clients cannot initialize.

