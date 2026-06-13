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
  LayoutDashboard,
  Activity,
  Key,
  FileText,
  Wallet,
  Box,
  Users,
  Ticket,
  User,
  Command,
  Radio,
  FlaskConical,
  MessageSquare,
  CreditCard,
  ListTodo,
  Settings,
  Building2,
  Network,
  GitBranch,
  Share2,
  ReceiptText,
  BarChart3,
  Gift,
  Landmark,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { useAuthStore } from '@/stores/auth-store'
import { WORKSPACE_IDS } from '@/components/layout/lib/workspace-registry'
import { type SidebarData } from '@/components/layout/types'
import { getUserRoleKey, ROLE_KEY } from '@/lib/roles'

export function useSidebarData(): SidebarData {
  const { t } = useTranslation()
  const user = useAuthStore((state) => state.auth.user)
  const roleKey = getUserRoleKey(user)

  const workspaces = [
    {
      id: WORKSPACE_IDS.DEFAULT,
      name: '', // Dynamically fetches system name
      logo: Command,
      plan: '', // Dynamically fetches system version
    },
  ]

  if (roleKey === ROLE_KEY.TENANT_ADMIN) {
    return {
      workspaces,
      navGroups: [
        {
          id: 'tenant-admin',
          title: t('Tenant Admin Console'),
          items: [
            {
              title: t('Tenant Console'),
              url: '/dashboard/overview',
              icon: LayoutDashboard,
            },
            {
              title: t('Enterprise Management'),
              icon: Building2,
              items: [
                {
                  title: t('User Management'),
                  url: '/users',
                },
                {
                  title: t('Channel Management'),
                  url: '/channels',
                },
              ],
            },
            {
              title: t('Commercial Center'),
              icon: CreditCard,
              items: [
                {
                  title: t('Subscription Management'),
                  url: '/subscriptions',
                },
                {
                  title: t('Billing Center'),
                  url: '/billing-dashboard',
                },
                {
                  title: t('Voucher Management'),
                  url: '/admin/vouchers',
                },
                {
                  title: t('Invoice Management'),
                  url: '/admin/invoices',
                },
              ],
            },
            {
              title: t('Operations Center'),
              icon: BarChart3,
              items: [
                {
                  title: t('Usage Logs'),
                  url: '/usage-logs/common',
                },
                {
                  title: t('Quota and Usage'),
                  url: '/quota-dashboard',
                  activeUrls: ['/usage-analytics'],
                },
                {
                  title: t('Usage Analytics'),
                  url: '/usage-analytics',
                },
                {
                  title: t('Payment Center'),
                  url: '/payment-center',
                },
                {
                  title: t('Revenue Share Dashboard'),
                  url: '/revenue-share',
                },
              ],
            },
            {
              title: t('Enterprise Settings'),
              url: '/profile',
              icon: Settings,
            },
          ],
        },
      ],
    }
  }

  return {
    workspaces,
    navGroups: [
      {
        id: 'chat',
        title: t('Chat'),
        items: [
          {
            title: t('Playground'),
            url: '/playground',
            icon: FlaskConical,
          },
          {
            title: t('Chat'),
            icon: MessageSquare,
            type: 'chat-presets',
          },
        ],
      },
      {
        id: 'general',
        title: t('General'),
        items: [
          {
            title: t('Overview'),
            url: '/dashboard/overview',
            icon: Activity,
          },
          {
            title: t('Dashboard'),
            url: '/dashboard/models',
            icon: LayoutDashboard,
          },
          {
            title: t('API Keys'),
            url: '/keys',
            icon: Key,
          },
          {
            title: t('Usage Logs'),
            url: '/usage-logs/common',
            icon: FileText,
          },
          {
            title: t('Task Logs'),
            url: '/usage-logs/task',
            activeUrls: ['/usage-logs/drawing'],
            configUrls: ['/usage-logs/drawing', '/usage-logs/task'],
            icon: ListTodo,
          },
        ],
      },
      {
        id: 'personal',
        title: t('Personal'),
        items: [
          {
            title: t('Wallet'),
            url: '/wallet',
            icon: Wallet,
          },
          {
            title: t('Billing'),
            url: '/billing',
            icon: ReceiptText,
          },
          {
            title: t('Voucher'),
            url: '/vouchers',
            icon: Gift,
          },
          {
            title: t('Invoice'),
            url: '/invoices',
            icon: ReceiptText,
          },
          {
            title: t('Profile'),
            url: '/profile',
            icon: User,
          },
        ],
      },
      {
        id: 'admin',
        title: t('Admin'),
        items: [
          {
            title: t('Tenants'),
            url: '/tenants',
            icon: Building2,
          },
          {
            title: t('Organizations'),
            url: '/organizations',
            icon: Network,
          },
          {
            title: t('Departments'),
            url: '/departments',
            icon: GitBranch,
          },
          {
            title: t('Distribution Channels'),
            url: '/distribution-channels',
            icon: Share2,
          },
          {
            title: t('Channels'),
            url: '/channels',
            icon: Radio,
          },
          {
            title: t('Models'),
            url: '/models/metadata',
            icon: Box,
          },
          {
            title: t('Users'),
            url: '/users',
            icon: Users,
          },
          {
            title: t('TopUp'),
            url: '/topup',
            icon: ReceiptText,
          },
          {
            title: t('Redemption Codes'),
            url: '/redemption-codes',
            icon: Ticket,
          },
          {
            title: t('Voucher'),
            url: '/admin/vouchers',
            icon: Gift,
          },
          {
            title: t('Finance'),
            url: '/admin/finance',
            icon: Landmark,
          },
          {
            title: t('Invoice Management'),
            url: '/admin/invoices',
            icon: ReceiptText,
          },
          {
            title: t('Subscription Management'),
            url: '/subscriptions',
            icon: CreditCard,
          },
          {
            title: t('Logs'),
            url: '/usage-logs/common',
            activeUrls: ['/usage-logs'],
            icon: FileText,
          },
          {
            title: t('Statistics'),
            url: '/dashboard/models',
            activeUrls: ['/dashboard/models'],
            icon: BarChart3,
          },
          {
            title: t('System Settings'),
            url: '/system-settings/site',
            activeUrls: ['/system-settings'],
            icon: Settings,
          },
        ],
      },
    ],
  }
}
