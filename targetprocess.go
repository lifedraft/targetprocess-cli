// Package targetprocess provides a typed Go client for the Targetprocess REST API.
//
// The package offers three layers of type safety:
//
//   - Generic functions with known entity types for full compile-time safety
//   - Generic functions with user-defined structs for typed v2 queries
//   - Untyped Client methods returning map[string]any for dynamic use
//
// Quick start:
//
//	c, err := targetprocess.NewClient("yourcompany.tpondemand.com", "your-token")
//	story, err := targetprocess.Get[targetprocess.UserStory](ctx, c, 12345)
//	fmt.Println(story.Name, story.EntityState.Name)
package targetprocess

// Identifiable is implemented by any entity that has a numeric ID.
type Identifiable interface {
	GetID() int
}

// Typed is implemented by entities that know their Targetprocess resource type.
// Generic functions use this to infer the API path automatically.
type Typed interface {
	Identifiable
	TPResourceType() string
}

// Entity represents a generic TP entity as a flexible map (untyped layer).
type Entity = map[string]any

// Ref is a lightweight reference to a related entity.
type Ref struct {
	ID           int    `json:"Id"`
	Name         string `json:"Name,omitempty"`
	ResourceType string `json:"ResourceType,omitempty"`
}

// UserRef is a reference to a Targetprocess user.
type UserRef struct {
	ID           int    `json:"Id"`
	FirstName    string `json:"FirstName,omitempty"`
	LastName     string `json:"LastName,omitempty"`
	FullName     string `json:"FullName,omitempty"`
	Login        string `json:"Login,omitempty"`
	ResourceType string `json:"ResourceType,omitempty"`
}

// EntityStateRef is a reference to an entity's workflow state.
type EntityStateRef struct {
	ID              int    `json:"Id"`
	Name            string `json:"Name,omitempty"`
	NumericPriority int    `json:"NumericPriority,omitempty"`
	ResourceType    string `json:"ResourceType,omitempty"`
}

// PriorityRef is a reference to a priority level.
type PriorityRef struct {
	ID           int    `json:"Id"`
	Name         string `json:"Name,omitempty"`
	Importance   int    `json:"Importance,omitempty"`
	ResourceType string `json:"ResourceType,omitempty"`
}

// CustomField represents a custom field value on an entity.
type CustomField struct {
	Name  string `json:"Name"`
	Type  string `json:"Type,omitempty"`
	Value any    `json:"Value"`
}

// BaseEntity contains fields common to most Targetprocess work items.
// Known entity types embed this struct.
type BaseEntity struct {
	ID                  int             `json:"Id"`
	Name                string          `json:"Name,omitempty"`
	Description         string          `json:"Description,omitempty"`
	ResourceType        string          `json:"ResourceType,omitempty"`
	EntityState         *EntityStateRef `json:"EntityState,omitempty"`
	EntityType          *Ref            `json:"EntityType,omitempty"`
	Project             *Ref            `json:"Project,omitempty"`
	Priority            *PriorityRef    `json:"Priority,omitempty"`
	Owner               *UserRef        `json:"Owner,omitempty"`
	Creator             *UserRef        `json:"Creator,omitempty"`
	LastEditor          *UserRef        `json:"LastEditor,omitempty"`
	Team                *Ref            `json:"Team,omitempty"`
	TeamIteration       *Ref            `json:"TeamIteration,omitempty"`
	Iteration           *Ref            `json:"Iteration,omitempty"`
	Release             *Ref            `json:"Release,omitempty"`
	ResponsibleTeam     *Ref            `json:"ResponsibleTeam,omitempty"`
	Tags                string          `json:"Tags,omitempty"`
	CreateDate          string          `json:"CreateDate,omitempty"`
	ModifyDate          string          `json:"ModifyDate,omitempty"`
	StartDate           string          `json:"StartDate,omitempty"`
	EndDate             string          `json:"EndDate,omitempty"`
	PlannedStartDate    string          `json:"PlannedStartDate,omitempty"`
	PlannedEndDate      string          `json:"PlannedEndDate,omitempty"`
	LastStateChangeDate string          `json:"LastStateChangeDate,omitempty"`
	Effort              float64         `json:"Effort,omitempty"`
	EffortCompleted     float64         `json:"EffortCompleted,omitempty"`
	EffortToDo          float64         `json:"EffortToDo,omitempty"`
	TimeSpent           float64         `json:"TimeSpent,omitempty"`
	TimeRemain          float64         `json:"TimeRemain,omitempty"`
	Progress            float64         `json:"Progress,omitempty"`
	NumericPriority     float64         `json:"NumericPriority,omitempty"`
	EntityVersion       int64           `json:"EntityVersion,omitempty"`
	Units               string          `json:"Units,omitempty"`
	CustomFields        []CustomField   `json:"CustomFields,omitempty"`
}

// GetID returns the entity's numeric ID.
func (e BaseEntity) GetID() int { return e.ID }

// UserStory represents a Targetprocess user story.
type UserStory struct {
	BaseEntity
	Feature *Ref `json:"Feature,omitempty"`
}

// TPResourceType returns the canonical Targetprocess resource type name.
func (UserStory) TPResourceType() string { return "UserStory" }

// Bug represents a Targetprocess bug.
type Bug struct {
	BaseEntity
	Severity  *Ref `json:"Severity,omitempty"`
	UserStory *Ref `json:"UserStory,omitempty"`
}

// TPResourceType returns the canonical Targetprocess resource type name.
func (Bug) TPResourceType() string { return "Bug" }

// Task represents a Targetprocess task.
type Task struct {
	BaseEntity
	UserStory *Ref `json:"UserStory,omitempty"`
}

// TPResourceType returns the canonical Targetprocess resource type name.
func (Task) TPResourceType() string { return "Task" }

// Feature represents a Targetprocess feature.
type Feature struct {
	BaseEntity
	Epic *Ref `json:"Epic,omitempty"`
}

// TPResourceType returns the canonical Targetprocess resource type name.
func (Feature) TPResourceType() string { return "Feature" }

// Epic represents a Targetprocess epic.
type Epic struct {
	BaseEntity
}

// TPResourceType returns the canonical Targetprocess resource type name.
func (Epic) TPResourceType() string { return "Epic" }

// Request represents a Targetprocess request.
type Request struct {
	BaseEntity
}

// TPResourceType returns the canonical Targetprocess resource type name.
func (Request) TPResourceType() string { return "Request" }

// Comment represents a Targetprocess comment on an entity.
type Comment struct {
	ID           int      `json:"Id"`
	Description  string   `json:"Description,omitempty"`
	CreateDate   string   `json:"CreateDate,omitempty"`
	Owner        *UserRef `json:"Owner,omitempty"`
	General      *Ref     `json:"General,omitempty"`
	ResourceType string   `json:"ResourceType,omitempty"`
}

// GetID returns the comment's numeric ID.
func (c Comment) GetID() int { return c.ID }

// TPResourceType returns the canonical Targetprocess resource type name.
func (Comment) TPResourceType() string { return "Comment" }
