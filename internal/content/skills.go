package content

import (
	"academy/internal/competency"
	"fmt"
)

func validateExerciseSkills(exercise Exercise) error {
	if exercise.Kind == "function" && len(exercise.Skills) == 0 {
		return fmt.Errorf("exercise %s has no competency mapping", exercise.ID)
	}
	if err := competency.Validate(exercise.Skills); err != nil {
		return fmt.Errorf("exercise %s: %w", exercise.ID, err)
	}
	return nil
}
