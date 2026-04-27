package models

import (
	"time"

	"github.com/lib/pq"
	"github.com/uptrace/bun"
)

type User struct {
    bun.BaseModel `bun:"table:users,alias:u"`

    ID        int64     `bun:",pk,autoincrement"`
    Name      string    `bun:",notnull"`
    Email     string    `bun:",unique,notnull"`
	Password  string    
    CreatedAt time.Time `bun:",nullzero,notnull,default:current_timestamp" json:"created_at"`
    UpdatedAt time.Time `bun:",nullzero,notnull,default:current_timestamp" json:"updated_at"`
}

type PublishStatusEnum string

const (
	PublishEnum_Draft   PublishStatusEnum="draft"
	PublishEnum_Published  PublishStatusEnum="published"
)

type VideoUpload struct {
	bun.BaseModel `bun:"table:video_uploads,alias:vdu"`

	ID		int64		`bun:",pk,autoincrement" json:"id"`
	UserID    int64     `bun:"user_id,notnull" json:"user_id"`
	Title         string    `bun:"title" json:"title"`
	Description   string    `bun:"description" json:"description"`
	Tags        pq.StringArray `bun:"tags,array" json:"tags"`   
	FileURL string 		`bun:"file_url,notnull" json:"file_url"`
	FileType string   `bun:"file_type,notnull" json:"file_type"`
	Thumbnails         []string `bun:"thumbnails,type:text[],notnull" json:"thumbnails"`
    SelectedThumbnail  string   `bun:"selected_thumbnail,type:text" json:"selected_thumbnail"`
	LikesCount    int64     `bun:",notnull" json:"likes_count"`
	ViewsCount    int64     `bun:",notnull" json:"views_count"`
	Duration      float64   `bun:"video_duration" json:"video_duration"`
	Category      string   `bun:",notnull" json:"category"`
	PublishStatus PublishStatusEnum `bun:"publish_status,notnull,default:'draft'" json:"publish_status"`// draft | published
	IsDeleted     bool   `bun:"," json:"is_deleted"`
	SearchVector  string `bun:"search_vector" json:"search_vector"`
	TranscodeStatus bool `bun:"," json:"transcode_status"`
	CreatedAt time.Time `bun:",nullzero,notnull,default:current_timestamp" json:"created_at"`
    UpdatedAt time.Time `bun:",nullzero,notnull,default:current_timestamp" json:"updated_at"`

}

type VideoQuality struct {
    bun.BaseModel `bun:"table:video_qualities"`

    ID            int64  `bun:",pk,autoincrement" json:"id"`
    VideoID       int64  `bun:"video_upload_id,notnull" json:"video_upload_id"`          // FK
	UserID		  int64   `bun:",notnull" json:"user_id"`
    Quality       string `bun:"type:varchar(10)" json:"quality"`   // "1080p"
    Codec         string `bun:"type:varchar(20)" json:"codec"`   // "h264"
    Bitrate       string    `bun:",notnull" json:"bitrate"`            // 4000 (kbps)
    Resolution    string `bun:"type:varchar(20)" json:"resolution"`   // "1920x1080"
    PlaylistKey   string `bun:"type:text" json:"playlist_key"`           // S3 key
    CDNUrl        string `bun:"type:text" json:"cdn_url"`           // CloudFront URL
	Thumbnail     string `bun:"type:text" json:"thumbnail"`
    Status        string `bun:"type:varchar(20)" json:"status"`   // "ready"
    FileSizeBytes int64   `bun:"," json:"file_size_bytes"`                           // for analytics
    CreatedAt     time.Time
	IsDeleted    bool    `bun:"," json:"is_deleted"`

	// relation back to VideoUpload
     VideoUpload *VideoUpload `bun:"rel:belongs-to,join:video_upload_id=id" json:"video_uploads,omitempty"`
}

type Clip struct {
	bun.BaseModel `bun:"table:clips,alias:c"`

	// Primary Keys
	ID             int64     `bun:",pk,autoincrement" json:"id"`
	VideoUploadID  int64     `bun:",notnull" json:"video_upload_id"`
	UserID         int64     `bun:",notnull" json:"user_id"`
	
	// Clip Details
	Title          string    `bun:"type:varchar(255)" json:"title"`
	Description    string    `bun:"type:text" json:"description,omitempty"`
	
	// Timing (in seconds)
	StartTime      float64   `bun:",notnull" json:"start_time"`              // 12.5 seconds
	EndTime        float64   `bun:",notnull" json:"end_time"`                // 37.8 seconds
	Duration       int       `json:"duration"`                                // calculated: end - start
	
	// Caption/Subtitle
	CaptionText    string    `bun:"type:text" json:"caption_text,omitempty"`
	TranscriptSnippet string `bun:"type:text" json:"transcript_snippet,omitempty"`
	
	// Platform-Specific URLs (JSON for flexibility)
	GeneratedFiles map[string]string `bun:"type:jsonb" json:"generated_files,omitempty"`
	// Example: { "tiktok": "s3://...", "youtube_shorts": "s3://...", "instagram_reels": "s3://..." }
	
	// Thumbnails
	ThumbnailURL   string    `bun:"type:text" json:"thumbnail_url,omitempty"`
	
	// Quality Metrics (optional for Phase 2)
	ViralityScore  *int      `json:"virality_score,omitempty"`               // 0-100
	
	// Processing
	Status         string    `bun:"type:varchar(50),default:'pending'" json:"status"`
	// Status values: pending, generating, completed, failed
	
	ProcessingError string   `bun:"type:text" json:"processing_error,omitempty"`
	
	// Platform Settings (what user selected)
	TargetPlatforms []string `bun:"type:jsonb" json:"target_platforms"`
	// Example: ["tiktok", "youtube_shorts", "instagram_reels"]
	
	// Timestamps
	CreatedAt      time.Time  `bun:",nullzero,notnull,default:current_timestamp" json:"created_at"`
	CompletedAt    *time.Time `bun:",nullzero" json:"completed_at,omitempty"`
	
	// Relations
	VideoUpload    *VideoQuality `bun:"rel:belongs-to,join:videodetails_upload_id=id" json:"videodetails_upload,omitempty"`
	User           *User        `bun:"rel:belongs-to,join:user_id=id" json:"user,omitempty"`
}

type CreatorSystem struct{
    ID            int64  `bun:",pk,autoincrement" json:"id"`
	UserID		  int64   `bun:",notnull" json:"user_id"`
	Followers     int64   `bun:"," json:"followers"`
	Niche 		  int64   `bun:"," json:"niche"`
	Earnings      float64 `bun:"," json:"earnings"`
	Ratings 	  int     `bun:"," json:"ratings"`
	IsDeleted    bool    `bun:"," json:"is_deleted"`
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

	Status CommentStatus `bun:"status,type:varchar(20),notnull,default:'VISIBLE'" json:"status"`

	CreatedAt time.Time  `bun:"created_at,nullzero,notnull,default:current_timestamp" json:"created_at"`
	UpdatedAt time.Time  `bun:"updated_at,nullzero,notnull,default:current_timestamp" json:"updated_at"`
	DeletedAt *time.Time `bun:"deleted_at,soft_delete,nullzero" json:"deleted_at,omitempty"`

	User *User `bun:"rel:belongs-to,join:user_id=id" json:"user,omitempty"`
}






// authentication  and user
// creator system
// business system
// video content system
// post/feed
// review
// product
// search
// recommendation
// shorts
// location
// notification 
// analytics