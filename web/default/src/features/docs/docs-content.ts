/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import {
  BookOpen,
  CircleHelp,
  CreditCard,
  FileKey,
  FileText,
  KeyRound,
  LayoutDashboard,
  ReceiptText,
  Ticket,
  type LucideIcon,
} from 'lucide-react'

export type UserGuideDoc = {
  slug: string
  titleKey: string
  descriptionKey: string
  markdownPath: string
  icon: LucideIcon
  sections: {
    headingKey: string
    bodyKey: string
  }[]
}

export const userGuideDocs: UserGuideDoc[] = [
  {
    slug: 'quick-start',
    titleKey: 'Quick Start',
    descriptionKey: 'Set up Quinta AI Gateway and complete the first request.',
    markdownPath: 'docs/user-guide/quick_start.md',
    icon: BookOpen,
    sections: [
      {
        headingKey: 'Prepare your account',
        bodyKey:
          'Sign in to Quinta AI Gateway, confirm your tenant access, and make sure your account has available quota before creating an API key.',
      },
      {
        headingKey: 'Create the first API key',
        bodyKey:
          'Open the API key page, create a key for your application, copy it once, and store it in a secure server-side environment variable.',
      },
      {
        headingKey: 'Send a test request',
        bodyKey:
          'Use the unified gateway endpoint with your API key, select an enabled model, and verify that the response and usage record are generated.',
      },
    ],
  },
  {
    slug: 'api-access',
    titleKey: 'API Access Guide',
    descriptionKey: 'Connect applications through the unified gateway API.',
    markdownPath: 'docs/user-guide/api_access.md',
    icon: FileKey,
    sections: [
      {
        headingKey: 'Use the unified endpoint',
        bodyKey:
          'Applications should call the Quinta AI Gateway endpoint instead of provider-specific endpoints so routing, quota, billing, and audit policies stay centralized.',
      },
      {
        headingKey: 'Pass authentication safely',
        bodyKey:
          'Send the API key in the authorization header from server-side code only. Do not expose keys in browsers, mobile bundles, or public repositories.',
      },
      {
        headingKey: 'Check request results',
        bodyKey:
          'After a request completes, review usage logs for status, latency, model, provider, token usage, and billing alignment.',
      },
    ],
  },
  {
    slug: 'account-and-api-key',
    titleKey: 'Account and API Key',
    descriptionKey: 'Manage accounts, API keys, permissions, and key safety.',
    markdownPath: 'docs/user-guide/account_and_api_key.md',
    icon: KeyRound,
    sections: [
      {
        headingKey: 'Account responsibility',
        bodyKey:
          'Keep account credentials private, enable available security options, and ask an administrator to adjust permissions when your role changes.',
      },
      {
        headingKey: 'API key lifecycle',
        bodyKey:
          'Create separate keys for different applications, rotate them regularly, and disable keys immediately when a project is retired.',
      },
      {
        headingKey: 'Permission boundaries',
        bodyKey:
          'Tenant and administrator pages are controlled by role permissions. If a page returns 403, contact your tenant administrator instead of retrying with another URL.',
      },
    ],
  },
  {
    slug: 'quota-and-billing',
    titleKey: 'Quota and Billing',
    descriptionKey:
      'Understand quota balance, usage metering, and billing records.',
    markdownPath: 'docs/user-guide/quota_and_billing.md',
    icon: CreditCard,
    sections: [
      {
        headingKey: 'Quota states',
        bodyKey:
          'Quota can be available, frozen, consumed, or adjusted by operations such as top-up, package assignment, refund, or manual review.',
      },
      {
        headingKey: 'Usage metering',
        bodyKey:
          'Requests are measured by model, provider, input tokens, output tokens, total tokens, and settlement rules configured by the platform.',
      },
      {
        headingKey: 'Billing review',
        bodyKey:
          'Use billing dashboards and usage logs together to reconcile daily spend, monthly spend, cumulative spend, and model-level cost ranking.',
      },
    ],
  },
  {
    slug: 'subscription',
    titleKey: 'Subscription Management',
    descriptionKey:
      'Use plans and subscriptions to manage recurring service rights.',
    markdownPath: 'docs/user-guide/subscription.md',
    icon: ReceiptText,
    sections: [
      {
        headingKey: 'Choose a plan',
        bodyKey:
          'Plans define recurring service rights such as quota, period, model access, and tenant-level commercial policies.',
      },
      {
        headingKey: 'Track subscription status',
        bodyKey:
          'Review whether a subscription is active, expired, pending payment, or under administrative review before relying on the included rights.',
      },
      {
        headingKey: 'Coordinate with billing',
        bodyKey:
          'Subscription changes should be checked against billing records and quota changes so user rights and financial records remain aligned.',
      },
    ],
  },
  {
    slug: 'voucher',
    titleKey: 'Voucher Management',
    descriptionKey: 'Issue, redeem, and audit quota or subscription vouchers.',
    markdownPath: 'docs/user-guide/voucher.md',
    icon: Ticket,
    sections: [
      {
        headingKey: 'Voucher types',
        bodyKey:
          'Vouchers can grant quota, subscription rights, or promotional benefits depending on administrator configuration.',
      },
      {
        headingKey: 'Redeem carefully',
        bodyKey:
          'Check the voucher validity period, usage limits, and tenant scope before redemption. Expired or already used vouchers cannot grant benefits.',
      },
      {
        headingKey: 'Audit changes',
        bodyKey:
          'Administrators should review voucher issuance, redemption, and quota changes to keep promotional operations traceable.',
      },
    ],
  },
  {
    slug: 'invoice',
    titleKey: 'Invoice Management',
    descriptionKey: 'Track invoice applications, statuses, and billing alignment.',
    markdownPath: 'docs/user-guide/invoice.md',
    icon: FileText,
    sections: [
      {
        headingKey: 'Submit invoice requests',
        bodyKey:
          'Create invoice requests from eligible billing records and provide the required billing title, tax information, and contact details.',
      },
      {
        headingKey: 'Review invoice status',
        bodyKey:
          'Invoice records may be pending, approved, rejected, or completed. Rejected records should include a reason for correction.',
      },
      {
        headingKey: 'Match invoices with bills',
        bodyKey:
          'Use invoice records together with billing records to verify amount, period, payer, and tenant ownership.',
      },
    ],
  },
  {
    slug: 'admin-console',
    titleKey: 'Admin Console',
    descriptionKey:
      'Operate tenant users, channels, billing, usage, and settings.',
    markdownPath: 'docs/user-guide/admin_console.md',
    icon: LayoutDashboard,
    sections: [
      {
        headingKey: 'Tenant operations',
        bodyKey:
          'Tenant administrators can manage users, channels, subscriptions, billing, vouchers, invoices, usage logs, quota, payment records, revenue share, and enterprise settings.',
      },
      {
        headingKey: 'Permission-first navigation',
        bodyKey:
          'Visible menus should match role permissions. Direct URL access to unauthorized pages should return 403 instead of rendering partial data.',
      },
      {
        headingKey: 'Operational review',
        bodyKey:
          'Before changing billing, quota, or channel settings, review the affected tenant, user scope, and audit trail.',
      },
    ],
  },
  {
    slug: 'faq',
    titleKey: 'FAQ',
    descriptionKey: 'Review common questions about access, billing, and operations.',
    markdownPath: 'docs/user-guide/faq.md',
    icon: CircleHelp,
    sections: [
      {
        headingKey: 'Why do I see 403?',
        bodyKey:
          'A 403 page means your current role does not have permission for that route. Ask an administrator to review your role and tenant scope.',
      },
      {
        headingKey: 'Why did a request consume quota?',
        bodyKey:
          'Quota consumption depends on model pricing, token usage, provider settlement, and configured billing policies. Check the usage log and billing dashboard together.',
      },
      {
        headingKey: 'Where should documentation links go?',
        bodyKey:
          'Public documentation entries should stay inside Quinta AI Gateway routes. Legacy external documentation links are normalized to the internal documentation center.',
      },
    ],
  },
]

export function getUserGuideDoc(slug: string): UserGuideDoc | undefined {
  return userGuideDocs.find((doc) => doc.slug === slug)
}
