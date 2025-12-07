package models

import "time"

// FeedbackCategory represents the type of feedback.
type FeedbackCategory string

const (
	FeedbackCategoryBug           FeedbackCategory = "bug"
	FeedbackCategoryFeature       FeedbackCategory = "feature"
	FeedbackCategoryInconvenience FeedbackCategory = "inconvenience"
	FeedbackCategoryOther         FeedbackCategory = "other"
)

// FeedbackUserInfo contains user information from the feedback form.
type FeedbackUserInfo struct {
	UserID    string `json:"userId,omitempty"`
	Email     string `json:"email,omitempty"`
	FirstName string `json:"firstName,omitempty"`
	LastName  string `json:"lastName,omitempty"`
}

// FeedbackEnvironment contains environment information from the feedback form.
type FeedbackEnvironment struct {
	UserAgent  string `json:"userAgent,omitempty"`
	ScreenSize string `json:"screenSize,omitempty"`
	URL        string `json:"url,omitempty"`
	PWA        bool   `json:"pwa,omitempty"`
	OS         string `json:"os,omitempty"`
}

// Feedback represents a user feedback entry.
type Feedback struct {
	ID          string              `json:"id" db:"id"`
	Category    FeedbackCategory    `json:"category" db:"category"`
	Message     string              `json:"message" db:"message"`
	UserInfo    FeedbackUserInfo    `json:"userInfo" db:"-"`
	Environment FeedbackEnvironment `json:"environment" db:"-"`
	// UserID is set from JWT if authenticated, otherwise from userInfo.userId
	UserID    *string   `json:"userId,omitempty" db:"user_id"`
	IsRead    bool      `json:"isRead" db:"is_read"`
	CreatedAt time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt time.Time `json:"updatedAt" db:"updated_at"`
	// For DB storage (JSONB)
	UserInfoJSON    []byte `json:"-" db:"user_info"`
	EnvironmentJSON []byte `json:"-" db:"environment"`
}

// CreateFeedbackRequest is the request body for creating feedback.
type CreateFeedbackRequest struct {
	Category    FeedbackCategory    `json:"category" binding:"required"`
	Message     string              `json:"message" binding:"required"`
	UserInfo    FeedbackUserInfo    `json:"userInfo"`
	Environment FeedbackEnvironment `json:"environment"`
}

// FeedbackListResponse is the response for listing feedback with pagination.
type FeedbackListResponse struct {
	Feedbacks  []*Feedback `json:"feedbacks"`
	Total      int         `json:"total"`
	Page       int         `json:"page"`
	PageSize   int         `json:"pageSize"`
	UnreadOnly bool        `json:"unreadOnly"`
}

// MarkFeedbackReadRequest is the request body for marking feedback as read.
type MarkFeedbackReadRequest struct {
	IsRead bool `json:"isRead"`
}
