package types

// Tag represents a resource tag
type Tag struct {
	Key   *string `json:"key,omitempty"`
	Value *string `json:"value,omitempty"`
}

// Attribute represents an attribute
type Attribute struct {
	Name       *string `json:"name"`
	Value      *string `json:"value,omitempty"`
	TargetType *string `json:"targetType,omitempty"`
	TargetId   *string `json:"targetId,omitempty"`
}

// Attachment represents an attachment
type Attachment struct {
	Id      *string        `json:"id,omitempty"`
	Type    *string        `json:"type,omitempty"`
	Status  *string        `json:"status,omitempty"`
	Details []KeyValuePair `json:"details,omitempty"`
}

// Failure represents a failure response
type Failure struct {
	Arn    *string `json:"arn,omitempty"`
	Reason *string `json:"reason,omitempty"`
	Detail *string `json:"detail,omitempty"`
}

// CapacityStrategy represents a capacity provider strategy
type CapacityStrategy struct {
	CapacityProvider *string `json:"capacityProvider"`
	Weight           *int    `json:"weight,omitempty"`
	Base             *int    `json:"base,omitempty"`
}
