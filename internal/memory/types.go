package memory

import "time"

// Memory type constants
const (
	TypeEmployeeInsight = "employee_insight"
	TypeStrategyResult  = "strategy_result"
	TypeOrgKnowledge    = "org_knowledge"
)

// Memory tier constants
const (
	TierShortTerm = "short_term"
	TierLongTerm  = "long_term"
	TierProfile   = "profile"
)

// Source type constants
const (
	SourceReport       = "report"
	SourceChase        = "chase"
	SourceSummary      = "summary"
	SourceConversation = "conversation"
	SourceManual       = "manual"
)

// ConsolidationTask represents the type of periodic maintenance
type ConsolidationTask string

const (
	ConsolidationClean   ConsolidationTask = "clean"
	ConsolidationMerge   ConsolidationTask = "merge"
	ConsolidationRebuild ConsolidationTask = "rebuild"
)

// Memory represents a single memory record at the service layer.
// IDs are strings (matching existing codebase pattern); converted to pgtype at DB boundary.
type Memory struct {
	ID          string
	TenantID    string
	MemoryType  string
	MemoryTier  string
	EmployeeID  string
	SourceType  string
	SourceID    string
	Content     string
	Summary     string
	Embedding   []float32
	Importance  float64
	AccessCount int
	Metadata    map[string]any
	ExpiresAt   *time.Time
	MergedInto  string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Similarity  float64 // populated by search queries
}

// RecallQuery is the input for semantic memory retrieval.
type RecallQuery struct {
	TenantID   string
	EmployeeID string
	QueryText  string
	MaxResults int // default 5
	MaxTokens  int // default 800
}

// RecallResult is the output of memory retrieval, slotted by type.
type RecallResult struct {
	Profile    *Memory
	Insights   []Memory
	Strategies []Memory
	Knowledge  []Memory
	TokenCount int
}

// Input types for extraction (decoupled from internal/report types)
type ReportInput struct {
	TenantID   string
	EmployeeID string
	ReportID   string
	Content    string
}

type ChaseInput struct {
	TenantID   string
	EmployeeID string
	ChaseLogID string
	Step       int
	Action     string
	Message    string
	Response   string
}

type SummaryInput struct {
	TenantID  string
	SummaryID string
	Content   string
}
