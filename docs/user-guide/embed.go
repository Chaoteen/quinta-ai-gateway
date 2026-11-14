// Package userguide contains only the reviewed public end-user documentation.
package userguide

import "embed"

// Files deliberately excludes operational, security, private and arbitrary tenant files.
//go:embed quick_start.md api_access.md account_and_api_key.md quota_and_billing.md subscription.md voucher.md invoice.md faq.md
var Files embed.FS
