package models

import (
	"time"

	"github.com/lib/pq"
	"github.com/uptrace/bun"
)

type User struct {
	bun.BaseModel `bun:"table:users,alias:u"`

	ID              int64     `bun:",pk,autoincrement"`
	Name            string    `bun:",notnull"`
	Email           string    `bun:",unique,notnull"`
	Password        string    `bun:"password" json:"password,omitempty"`
	ConfirmPassword string    `bun:"-" json:"confirm_password,omitempty"`
	IsVerified      bool      `bun:"is_verified,default:false" json:"is_verified"`
	Avatar          string    `bun:"avatar,type:text" json:"avatar"`
	OTP             string    `bun:"otp" json:"otp"`
	OTPExpiry       time.Time `bun:"otp_expiry" json:"otp_expiry"`
	CreatedAt       time.Time `bun:",nullzero,notnull,default:current_timestamp" json:"created_at"`
	UpdatedAt       time.Time `bun:",nullzero,notnull,default:current_timestamp" json:"updated_at"`
}

type PublishStatusEnum string

const (
	PublishEnum_Draft     PublishStatusEnum = "draft"
	PublishEnum_Published PublishStatusEnum = "published"
)

type VideoUpload struct {
	bun.BaseModel `bun:"table:video_uploads,alias:vdu"`

	ID                int64             `bun:",pk,autoincrement" json:"id"`
	UserID            int64             `bun:"user_id,notnull" json:"user_id"`
	Title             string            `bun:"title" json:"title"`
	Description       string            `bun:"description" json:"description"`
	Tags              pq.StringArray    `bun:"tags,array" json:"tags"`
	FileURL           string            `bun:"file_url,notnull" json:"file_url"`
	FileType          string            `bun:"file_type,notnull" json:"file_type"`
	Thumbnails        []string          `bun:"thumbnails,type:text[],notnull" json:"thumbnails"`
	SelectedThumbnail string            `bun:"selected_thumbnail,type:text" json:"selected_thumbnail"`
	LikesCount        int64             `bun:",notnull" json:"likes_count"`
	ViewsCount        int64             `bun:",notnull" json:"views_count"`
	Duration          float64           `bun:"video_duration" json:"video_duration"`
	Category          string            `bun:",notnull" json:"category"`
	PublishStatus     PublishStatusEnum `bun:"publish_status,notnull,default:'draft'" json:"publish_status"` // draft | published
	MasterCdnUrl      string            `bun:"master_cdn_url,type:text" json:"master_cdn_url"`
	IsDeleted         bool              `bun:"," json:"is_deleted"`
	SearchVector      string            `bun:"search_vector" json:"search_vector"`
	TranscodeStatus   bool              `bun:"," json:"transcode_status"`

	// Spec: Location and Merchant connection relationships
	MerchantID    *int64 `bun:"merchant_id" json:"merchant_id,omitempty"`
	ShopID        *int64 `bun:"shop_id" json:"shop_id,omitempty"`
	ProductID     *int64 `bun:"product_id" json:"product_id,omitempty"`
	ShopProductID *int64 `bun:"shop_product_id" json:"shop_product_id,omitempty"`

	CreatedAt time.Time `bun:",nullzero,notnull,default:current_timestamp" json:"created_at"`
	UpdatedAt time.Time `bun:",nullzero,notnull,default:current_timestamp" json:"updated_at"`
}

type VideoQuality struct {
	bun.BaseModel `bun:"table:video_qualities"`

	ID            int64  `bun:",pk,autoincrement" json:"id"`
	VideoID       int64  `bun:"video_upload_id,notnull" json:"video_upload_id"` // FK
	UserID        int64  `bun:",notnull" json:"user_id"`
	Quality       string `bun:"type:varchar(10)" json:"quality"`    // "1080p"
	Codec         string `bun:"type:varchar(20)" json:"codec"`      // "h264"
	Bitrate       string `bun:",notnull" json:"bitrate"`            // 4000 (kbps)
	Resolution    string `bun:"type:varchar(20)" json:"resolution"` // "1920x1080"
	PlaylistKey   string `bun:"type:text" json:"playlist_key"`      // S3 key
	CDNUrl        string `bun:"type:text" json:"cdn_url"`           // CloudFront URL
	Thumbnail     string `bun:"type:text" json:"thumbnail"`
	Status        string `bun:"type:varchar(20)" json:"status"` // "ready"
	FileSizeBytes int64  `bun:"," json:"file_size_bytes"`       // for analytics
	CreatedAt     time.Time
	IsDeleted     bool `bun:"," json:"is_deleted"`

	// relation back to VideoUpload
	VideoUpload *VideoUpload `bun:"rel:belongs-to,join:video_upload_id=id" json:"video_uploads,omitempty"`
}

type Clip struct {
	bun.BaseModel `bun:"table:clips,alias:c"`

	// Primary Keys
	ID            int64 `bun:",pk,autoincrement" json:"id"`
	VideoUploadID int64 `bun:",notnull" json:"video_upload_id"`
	UserID        int64 `bun:",notnull" json:"user_id"`

	// Clip Details
	Title       string `bun:"type:varchar(255)" json:"title"`
	Description string `bun:"type:text" json:"description,omitempty"`

	// Timing (in seconds)
	StartTime float64 `bun:",notnull" json:"start_time"` // 12.5 seconds
	EndTime   float64 `bun:",notnull" json:"end_time"`   // 37.8 seconds
	Duration  int     `json:"duration"`                  // calculated: end - start

	// Caption/Subtitle
	CaptionText       string `bun:"type:text" json:"caption_text,omitempty"`
	TranscriptSnippet string `bun:"type:text" json:"transcript_snippet,omitempty"`

	// Platform-Specific URLs (JSON for flexibility)
	GeneratedFiles map[string]string `bun:"type:jsonb" json:"generated_files,omitempty"`
	// Example: { "tiktok": "s3://...", "youtube_shorts": "s3://...", "instagram_reels": "s3://..." }

	// Thumbnails
	ThumbnailURL string `bun:"type:text" json:"thumbnail_url,omitempty"`

	// Quality Metrics (optional for Phase 2)
	ViralityScore *int `json:"virality_score,omitempty"` // 0-100

	// Processing
	Status string `bun:"type:varchar(50),default:'pending'" json:"status"`
	// Status values: pending, generating, completed, failed

	ProcessingError string `bun:"type:text" json:"processing_error,omitempty"`

	// Platform Settings (what user selected)
	TargetPlatforms []string `bun:"type:jsonb" json:"target_platforms"`
	// Example: ["tiktok", "youtube_shorts", "instagram_reels"]

	// Timestamps
	CreatedAt   time.Time  `bun:",nullzero,notnull,default:current_timestamp" json:"created_at"`
	CompletedAt *time.Time `bun:",nullzero" json:"completed_at,omitempty"`

	// Relations
	VideoUpload *VideoQuality `bun:"rel:belongs-to,join:videodetails_upload_id=id" json:"videodetails_upload,omitempty"`
	User        *User         `bun:"rel:belongs-to,join:user_id=id" json:"user,omitempty"`
}

type CreatorSystem struct {
	ID        int64   `bun:",pk,autoincrement" json:"id"`
	UserID    int64   `bun:",notnull" json:"user_id"`
	Followers int64   `bun:"," json:"followers"`
	Niche     int64   `bun:"," json:"niche"`
	Earnings  float64 `bun:"," json:"earnings"`
	Ratings   int     `bun:"," json:"ratings"`
	IsDeleted bool    `bun:"," json:"is_deleted"`
}

type CommentStatus string

const (
	CommentVisible       CommentStatus = "VISIBLE"
	CommentPendingReview CommentStatus = "PENDING_REVIEW"
	CommentHidden        CommentStatus = "HIDDEN"
	CommentDeleted       CommentStatus = "DELETED"
)

type Comment struct {
	bun.BaseModel `bun:"table:comments,alias:c"`

	ID              int64  `bun:",pk,autoincrement" json:"id"`
	VideoID         int64  `bun:"video_id,notnull" json:"video_id"`
	UserID          int64  `bun:"user_id,notnull" json:"user_id"`
	ParentCommentID *int64 `bun:"parent_comment_id" json:"parent_comment_id,omitempty"`
	RootCommentID   int64  `bun:"root_comment_id" json:"root_comment_id"`

	Depth      int    `bun:"depth,notnull,default:0" json:"depth"`
	Content    string `bun:"content,type:text,notnull" json:"content"`
	LikeCount  int64  `bun:"like_count,notnull,default:0" json:"like_count"`
	ReplyCount int64  `bun:"reply_count,notnull,default:0" json:"reply_count"`
	Rating     int64  `bun:"rating,notnull,default:0" json:"rating"`

	Status CommentStatus `bun:"status,type:varchar(20),notnull,default:'VISIBLE'" json:"status"`

	CreatedAt time.Time  `bun:"created_at,nullzero,notnull,default:current_timestamp" json:"created_at"`
	UpdatedAt time.Time  `bun:"updated_at,nullzero,notnull,default:current_timestamp" json:"updated_at"`
	DeletedAt *time.Time `bun:"deleted_at,soft_delete,nullzero" json:"deleted_at,omitempty"`

	User *User `bun:"rel:belongs-to,join:user_id=id" json:"user,omitempty"`
}

// UserRole represents a role in the system
type RoleName string

const (
	RoleCustomer RoleName = "CUSTOMER"
	RoleMerchant RoleName = "MERCHANT"
	RoleCreator  RoleName = "CREATOR"
	RoleAdmin    RoleName = "ADMIN"
)

type Role struct {
	bun.BaseModel `bun:"table:roles,alias:r"`

	ID   int64    `bun:",pk,autoincrement" json:"id"`
	Name RoleName `bun:"name,unique,notnull" json:"name"`
}

type UserRole struct {
	bun.BaseModel `bun:"table:user_roles,alias:ur"`

	ID     int64 `bun:",pk,autoincrement" json:"id"`
	UserID int64 `bun:"user_id,notnull" json:"user_id"`
	RoleID int64 `bun:"role_id,notnull" json:"role_id"`

	Role *Role `bun:"rel:belongs-to,join:role_id=id" json:"role,omitempty"`
	User *User `bun:"rel:belongs-to,join:user_id=id" json:"user,omitempty"`
}

// VerificationStatus
type VerificationStatus string

const (
	VerificationPending  VerificationStatus = "PENDING"
	VerificationVerified VerificationStatus = "VERIFIED"
	VerificationRejected VerificationStatus = "REJECTED"
)

// BusinessType
type BusinessType string

const (
	BusinessTypeRetail  BusinessType = "RETAIL"
	BusinessTypeOnline  BusinessType = "ONLINE"
	BusinessTypeService BusinessType = "SERVICE"
)

type MerchantProfile struct {
	bun.BaseModel `bun:"table:merchant_profiles,alias:mp"`

	ID                  int64              `bun:",pk,autoincrement" json:"id"`
	OwnerUserID         int64              `bun:"owner_user_id,notnull" json:"owner_user_id"`
	BusinessName        string             `bun:"business_name,notnull" json:"business_name"`
	BusinessDescription string             `bun:"business_description,type:text" json:"business_description"`
	BusinessType        BusinessType       `bun:"business_type,notnull,default:'RETAIL'" json:"business_type"`
	VerificationStatus  VerificationStatus `bun:"verification_status,notnull,default:'PENDING'" json:"verification_status"`
	CreatedAt           time.Time          `bun:",nullzero,notnull,default:current_timestamp" json:"created_at"`
	UpdatedAt           time.Time          `bun:",nullzero,notnull,default:current_timestamp" json:"updated_at"`

	Owner *User `bun:"rel:belongs-to,join:owner_user_id=id" json:"owner,omitempty"`
}

type MerchantPermission string

const (
	MerchantPermissionOwner            MerchantPermission = "OWNER"
	MerchantPermissionManager          MerchantPermission = "MANAGER"
	MerchantPermissionContentManager   MerchantPermission = "CONTENT_MANAGER"
	MerchantPermissionInventoryManager MerchantPermission = "INVENTORY_MANAGER"
)

type MemberStatus string

const (
	MemberStatusActive    MemberStatus = "ACTIVE"
	MemberStatusSuspended MemberStatus = "SUSPENDED"
)

type MerchantMember struct {
	bun.BaseModel `bun:"table:merchant_members,alias:mm"`

	ID         int64              `bun:",pk,autoincrement" json:"id"`
	MerchantID int64              `bun:"merchant_id,notnull" json:"merchant_id"`
	UserID     int64              `bun:"user_id,notnull" json:"user_id"`
	Permission MerchantPermission `bun:"permission,notnull,default:'MANAGER'" json:"permission"`
	Status     MemberStatus       `bun:"status,notnull,default:'ACTIVE'" json:"status"`
	CreatedAt  time.Time          `bun:",nullzero,notnull,default:current_timestamp" json:"created_at"`

	Merchant *MerchantProfile `bun:"rel:belongs-to,join:merchant_id=id" json:"merchant,omitempty"`
	User     *User            `bun:"rel:belongs-to,join:user_id=id" json:"user,omitempty"`
}

type ShopStatus string

const (
	ShopStatusActive   ShopStatus = "ACTIVE"
	ShopStatusInactive ShopStatus = "INACTIVE"
)

type Shop struct {
	bun.BaseModel `bun:"table:shops,alias:s"`

	ID                 int64              `bun:",pk,autoincrement" json:"id"`
	MerchantID         int64              `bun:"merchant_id,notnull" json:"merchant_id"`
	Name               string             `bun:"name,notnull" json:"name"`
	Description        string             `bun:"description,type:text" json:"description"`
	Phone              string             `bun:"phone" json:"phone"`
	Address            string             `bun:"address,type:text,notnull" json:"address"`
	City               string             `bun:"city" json:"city"`
	State              string             `bun:"state" json:"state"`
	Pincode            string             `bun:"pincode" json:"pincode"`
	Country            string             `bun:"country,default:'India'" json:"country"`
	Latitude           float64            `bun:"-" json:"latitude"`
	Longitude          float64            `bun:"-" json:"longitude"`
	Status             ShopStatus         `bun:"status,notnull,default:'ACTIVE'" json:"status"`
	VerificationStatus VerificationStatus `bun:"verification_status,notnull,default:'PENDING'" json:"verification_status"`
	CreatedAt          time.Time          `bun:",nullzero,notnull,default:current_timestamp" json:"created_at"`
	UpdatedAt          time.Time          `bun:",nullzero,notnull,default:current_timestamp" json:"updated_at"`

	Merchant *MerchantProfile `bun:"rel:belongs-to,join:merchant_id=id" json:"merchant,omitempty"`
}

type Product struct {
	bun.BaseModel `bun:"table:products,alias:p"`

	ID             int64                  `bun:",pk,autoincrement" json:"id"`
	Name           string                 `bun:"name,notnull" json:"name"`
	Brand          string                 `bun:"brand" json:"brand"`
	Description    string                 `bun:"description,type:text" json:"description"`
	CategoryID     *int64                 `bun:"category_id" json:"category_id,omitempty"`
	Specifications map[string]interface{} `bun:"specifications,type:jsonb" json:"specifications,omitempty"`
	CreatedAt      time.Time              `bun:",nullzero,notnull,default:current_timestamp" json:"created_at"`
	UpdatedAt      time.Time              `bun:",nullzero,notnull,default:current_timestamp" json:"updated_at"`
}

type StockAvailability string

const (
	AvailabilityAvailable  StockAvailability = "AVAILABLE"
	AvailabilityUnknown    StockAvailability = "UNKNOWN"
	AvailabilityOutOfStock StockAvailability = "OUT_OF_STOCK"
)

type ShopProductStatus string

const (
	ShopProductStatusActive   ShopProductStatus = "ACTIVE"
	ShopProductStatusInactive ShopProductStatus = "INACTIVE"
)

type ShopProduct struct {
	bun.BaseModel `bun:"table:shop_products,alias:sp"`

	ID             int64             `bun:",pk,autoincrement" json:"id"`
	ShopID         int64             `bun:"shop_id,notnull" json:"shop_id"`
	ProductID      int64             `bun:"product_id,notnull" json:"product_id"`
	Price          float64           `bun:"price,notnull" json:"price"`
	Currency       string            `bun:"currency,default:'INR'" json:"currency"`
	Availability   StockAvailability `bun:"availability,notnull,default:'AVAILABLE'" json:"availability"`
	StockUpdatedAt time.Time         `bun:"stock_updated_at,nullzero,notnull,default:current_timestamp" json:"stock_updated_at"`
	Status         ShopProductStatus `bun:"status,notnull,default:'ACTIVE'" json:"status"`
	CreatedAt      time.Time         `bun:",nullzero,notnull,default:current_timestamp" json:"created_at"`
	UpdatedAt      time.Time         `bun:",nullzero,notnull,default:current_timestamp" json:"updated_at"`

	Shop    *Shop    `bun:"rel:belongs-to,join:shop_id=id" json:"shop,omitempty"`
	Product *Product `bun:"rel:belongs-to,join:product_id=id" json:"product,omitempty"`
}

type UserSavedLocation struct {
	bun.BaseModel `bun:"table:user_saved_locations,alias:usl"`

	ID        int64     `bun:",pk,autoincrement" json:"id"`
	UserID    int64     `bun:"user_id,notnull" json:"user_id"`
	Label     string    `bun:"label" json:"label"` // e.g. Home, Work
	Address   string    `bun:"address,type:text" json:"address"`
	City      string    `bun:"city" json:"city"`
	Pincode   string    `bun:"pincode" json:"pincode"`
	Latitude  float64   `bun:"-" json:"latitude"`
	Longitude float64   `bun:"-" json:"longitude"`
	CreatedAt time.Time `bun:",nullzero,notnull,default:current_timestamp" json:"created_at"`

	User *User `bun:"rel:belongs-to,join:user_id=id" json:"user,omitempty"`
}
