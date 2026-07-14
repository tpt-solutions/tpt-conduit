# /api — API layer (Phase 2)

Planned: GraphQL schema (queries/mutations for tickets, workflows, runs,
approvals) and gRPC service definitions mirroring core mutations, plus
single-tenant basic-auth / API-key middleware.

The engine in `/engine` already exposes the full durable execution surface
that this layer will wrap.
