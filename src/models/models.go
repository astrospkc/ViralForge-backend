package models

import (
	"time"

	"github.com/uptrace/bun"
)

type User struct {
    bun.BaseModel `bun:"table:users,alias:u"`

    ID        int64     `bun:",pk,autoincrement"`
    Name      string    `bun:",notnull"`
    Email     string    `bun:",unique,notnull"`
	Password  string    
    CreatedAt time.Time `bun:",nullzero,notnull,default:current_timestamp"`
    UpdatedAt time.Time `bun:",nullzero,notnull,default:current_timestamp"`
}


type VideoUpload struct {
	bun.BaseModel `bun:"table:video_uploads,alias:vu"`

	// Primary Keys
	ID        int64     `bun:",pk,autoincrement" json:"id"`
	UserID    int64     `bun:",notnull" json:"user_id"`
	
	// Original Video Information
	OriginalFilename string    `bun:",notnull" json:"original_filename"`
	OriginalURL      string    `bun:"type:text" json:"original_url"`           // S3/R2 URL
	FileSize         int64     `json:"file_size"`                              // in bytes
	MimeType         string    `bun:"type:varchar(100)" json:"mime_type"`      // video/mp4, etc.
	
	// Video Metadata
	DurationSeconds  int       `json:"duration_seconds"`                       // total duration
	Width            int       `json:"width"`                                  // original resolution
	Height           int       `json:"height"`
	FPS              float64   `json:"fps"`                                    // frames per second
	Codec            string    `bun:"type:varchar(50)" json:"codec"`           // h264, h265, etc.
	Bitrate          int       `json:"bitrate"`                                // kbps
	
	// Processing Status
	Status           string    `bun:"type:varchar(50),default:'uploaded'" json:"status"` 
	// Status values: uploaded, processing, analyzed, completed, failed
	
	ProcessingError  string    `bun:"type:text" json:"processing_error,omitempty"`
	
	// Analysis Results (JSON fields for flexibility)
	Metadata         map[string]interface{} `bun:"type:jsonb" json:"metadata,omitempty"`
	// Stores: { "scenes": [...], "transcript": "...", "suggested_clips": [...] }
	
	// Timestamps
	UploadedAt       time.Time `bun:",nullzero,notnull,default:current_timestamp" json:"uploaded_at"`
	ProcessedAt      *time.Time `bun:",nullzero" json:"processed_at,omitempty"`
	
	// Relations
	User             *User      `bun:"rel:belongs-to,join:user_id=id" json:"user,omitempty"`
	Clips            []*Clip    `bun:"rel:has-many,join:id=video_upload_id" json:"clips,omitempty"`
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
	VideoUpload    *VideoUpload `bun:"rel:belongs-to,join:video_upload_id=id" json:"video_upload,omitempty"`
	User           *User        `bun:"rel:belongs-to,join:user_id=id" json:"user,omitempty"`
}