package plan

// Equal compares complete provider-independent Plan meaning. Invalid Plans
// fail closed and are never equal, including two zero Plans.
func (p Plan) Equal(other Plan) bool {
	if !p.valid || !other.valid {
		return false
	}
	if p.name != other.name || p.goal != other.goal || len(p.conditions) != len(other.conditions) || len(p.tasks) != len(other.tasks) {
		return false
	}
	for i := range p.conditions {
		if p.conditions[i] != other.conditions[i] {
			return false
		}
	}
	for i := range p.tasks {
		if p.tasks[i] != other.tasks[i] {
			return false
		}
	}
	if (p.targetDate == nil) != (other.targetDate == nil) {
		return false
	}
	return p.targetDate == nil || *p.targetDate == *other.targetDate
}
