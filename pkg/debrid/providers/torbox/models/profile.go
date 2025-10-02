package models

import "time"

type TorboxProfile struct {
	Id                        int64     `json:"id"`
	AuthId                    string    `json:"auth_id"`
	CreatedAt                 string    `json:"created_at"`
	UpdatedAt                 string    `json:"updated_at"`
	Plan                      int       `json:"plan"`
	TotalDownloaded           int64     `json:"total_downloaded"`
	Customer                  string    `json:"customer"`
	IsSubscribed              bool      `json:"is_subscribed"`
	PremiumExpiresAt          time.Time `json:"premium_expires_at"`
	CooldownUntil             time.Time `json:"cooldown_until"`
	Email                     string    `json:"email"`
	UserReferral              string    `json:"user_referral"`
	BaseEmail                 string    `json:"base_email"`
	TotalBytesDownloaded      int64     `json:"total_bytes_downloaded"`
	TotalBytesUploaded        int64     `json:"total_bytes_uploaded"`
	TorrentsDownloaded        int64     `json:"torrents_downloaded"`
	WebDownloadsDownloaded    int64     `json:"web_downloads_downloaded"`
	UsenetDownloadsDownloaded int64     `json:"usenet_downloads_downloaded"`
	AdditionalConcurrentSlots int64     `json:"additional_concurrent_slots"`
	LongTermSeeding           bool      `json:"long_term_seeding"`
	LongTermStorage           bool      `json:"long_term_storage"`
	IsVendor                  bool      `json:"is_vendor"`
	VendorId                  any       `json:"vendor_id"`
	PurchasesReferred         int64     `json:"purchases_referred"`
}
