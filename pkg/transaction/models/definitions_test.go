package models

import (
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestLLMReferenceGetLogFields(t *testing.T) {
	t.Run("populates id and model when id is set", func(t *testing.T) {
		l := LLMReference{
			ResourceReference: ResourceReference{ID: "llm-1"},
			Model:             "gpt-4",
		}
		fields := l.GetLogFields(logrus.Fields{}, "llm")
		assert.Equal(t, "llm-1", fields["llm"])
		assert.Equal(t, "gpt-4", fields["modelName"])
	})

	t.Run("adds nothing when id is empty", func(t *testing.T) {
		l := LLMReference{Model: "gpt-4"}
		fields := l.GetLogFields(logrus.Fields{}, "llm")
		assert.NotContains(t, fields, "llm")
		assert.NotContains(t, fields, "modelName")
	})
}
