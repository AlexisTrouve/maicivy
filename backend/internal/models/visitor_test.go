package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestVisitor_HasAccessToAI(t *testing.T) {
	tests := []struct {
		name     string
		visitor  Visitor
		expected bool
	}{
		{
			name: "3+ visits grants access",
			visitor: Visitor{
				VisitCount:      3,
				ProfileDetected: ProfileTypeUnknown,
			},
			expected: true,
		},
		{
			name: "Recruiter profile grants access immediately",
			visitor: Visitor{
				VisitCount:      1,
				ProfileDetected: ProfileTypeRecruiter,
			},
			expected: true,
		},
		{
			name: "CTO profile grants access immediately",
			visitor: Visitor{
				VisitCount:      1,
				ProfileDetected: ProfileTypeCTO,
			},
			expected: true,
		},
		{
			name: "Developer with 2 visits does not have access",
			visitor: Visitor{
				VisitCount:      2,
				ProfileDetected: ProfileTypeDeveloper,
			},
			expected: false,
		},
		{
			name: "Unknown profile with 1 visit does not have access",
			visitor: Visitor{
				VisitCount:      1,
				ProfileDetected: ProfileTypeUnknown,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.visitor.HasAccessToAI()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestVisitor_IncrementVisit(t *testing.T) {
	visitor := Visitor{
		VisitCount: 1,
		LastVisit:  time.Now().Add(-24 * time.Hour),
	}

	visitor.IncrementVisit()

	assert.Equal(t, 2, visitor.VisitCount)
	assert.WithinDuration(t, time.Now(), visitor.LastVisit, 1*time.Second)
}

func TestVisitor_IsTargetProfile(t *testing.T) {
	tests := []struct {
		name     string
		profile  ProfileType
		expected bool
	}{
		{"Recruiter is target", ProfileTypeRecruiter, true},
		{"CTO is target", ProfileTypeCTO, true},
		{"Developer is not target", ProfileTypeDeveloper, false},
		{"Unknown is not target", ProfileTypeUnknown, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			visitor := Visitor{ProfileDetected: tt.profile}
			assert.Equal(t, tt.expected, visitor.IsTargetProfile())
		})
	}
}
