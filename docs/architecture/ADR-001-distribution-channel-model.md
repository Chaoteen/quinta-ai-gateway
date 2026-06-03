# ADR-001 Distribution Channel Model

## Status

Accepted

## Context

Quinta AI Gateway supports:

- Multi Tenant
- Organization
- Department
- Distribution Channel
- Subscription
- Billing
- Revenue Attribution

A design decision is required regarding the relationship between
Distribution Channel and organizational hierarchy.

## Decision

Distribution Channel is an independent business dimension.

It is NOT:

- a child of Tenant
- a child of Organization
- a child of Department

Organizational hierarchy:

Tenant
 └ Organization
      └ Department

Distribution Channel exists as a parallel attribution dimension.

## Ownership Model

Business entities may carry:

- tenant_id
- organization_id
- department_id
- distribution_channel_id

These dimensions are independent.

## Purpose

Tenant / Organization / Department:

- access control
- ownership
- governance
- administration

Distribution Channel:

- customer acquisition
- sales attribution
- partner reporting
- commission calculation
- revenue analysis

## Future Design

Subscription:
- distribution_channel_id

Billing:
- distribution_channel_id

TopUp:
- distribution_channel_id

Usage:
- distribution_channel_id

Revenue Sharing:
- distribution_channel_id

## Consequences

A customer organization may be associated with users originating from different distribution channels.

This is allowed and expected.

