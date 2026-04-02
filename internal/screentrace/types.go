package screentrace

import "time"

const (
	DefaultIntervalSeconds      = 15
	DefaultRetentionDays        = 7
	DefaultDigestIntervalMinute = 15
	DefaultSimilarityThreshold  = 4
	DefaultMaxImageDimension    = 1600
	DefaultJPEGQuality          = 72
)

type Settings struct {
	Enabled            bool   `json:"enabled"`
	IntervalSeconds    int    `json:"intervalSeconds"`
	RetentionDays      int    `json:"retentionDays"`
	VisionProfileID    string `json:"visionProfileId"`
	WriteDigestsToKB   bool   `json:"writeDigestsToKb"`
	DigestIntervalMins int    `json:"digestIntervalMins"`
}

func DefaultSettings() Settings {
	return Settings{
		Enabled:            false,
		IntervalSeconds:    DefaultIntervalSeconds,
		RetentionDays:      DefaultRetentionDays,
		VisionProfileID:    "",
		WriteDigestsToKB:   false,
		DigestIntervalMins: DefaultDigestIntervalMinute,
	}
}

func (s Settings) Normalize() Settings {
	if s.IntervalSeconds <= 0 {
		s.IntervalSeconds = DefaultIntervalSeconds
	}
	if s.RetentionDays <= 0 {
		s.RetentionDays = DefaultRetentionDays
	}
	if s.DigestIntervalMins <= 0 {
		s.DigestIntervalMins = DefaultDigestIntervalMinute
	}
	return s
}

type Record struct {
	ID             string    `json:"id"`
	CapturedAt     time.Time `json:"capturedAt"`
	ImagePath      string    `json:"imagePath"`
	ImageHash      string    `json:"imageHash"`
	Width          int       `json:"width"`
	Height         int       `json:"height"`
	DisplayIndex   int       `json:"displayIndex"`
	SceneSummary   string    `json:"sceneSummary"`
	VisibleText    []string  `json:"visibleText"`
	Apps           []string  `json:"apps"`
	TaskGuess      string    `json:"taskGuess"`
	Keywords       []string  `json:"keywords"`
	SensitiveLevel string    `json:"sensitiveLevel"`
	Confidence     float64   `json:"confidence"`
	CreatedAt      time.Time `json:"createdAt"`
}

type Digest struct {
	ID               string    `json:"id"`
	BucketStart      time.Time `json:"bucketStart"`
	BucketEnd        time.Time `json:"bucketEnd"`
	RecordCount      int       `json:"recordCount"`
	Summary          string    `json:"summary"`
	Keywords         []string  `json:"keywords"`
	DominantApps     []string  `json:"dominantApps"`
	DominantTasks    []string  `json:"dominantTasks"`
	WrittenToKB      bool      `json:"writtenToKb"`
	KnowledgeEntryID string    `json:"knowledgeEntryId"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

type Status struct {
	Settings          Settings  `json:"settings"`
	Running           bool      `json:"running"`
	LastCaptureAt     time.Time `json:"lastCaptureAt"`
	LastAnalysisAt    time.Time `json:"lastAnalysisAt"`
	LastDigestAt      time.Time `json:"lastDigestAt"`
	LastError         string    `json:"lastError"`
	LastImagePath     string    `json:"lastImagePath"`
	TotalRecords      int       `json:"totalRecords"`
	SkippedDuplicates int       `json:"skippedDuplicates"`
}

type Capture struct {
	DisplayIndex int
	Width        int
	Height       int
	ImageBytes   []byte
	ImageHash    string
}
