package opt

import (
	"reflect"

	gocmp "github.com/google/go-cmp/cmp"
)

// Path returns a path filter which can be used with
// github.com/google/go-cmp/cmp.FilerPath.
func Path(steps ...StepFilter) func(gocmp.Path) bool {
	return func(path gocmp.Path) bool {
		if len(path) != len(steps) {
			return false
		}
		for i, step := range path {
			if !steps[i](step) {
				return false
			}
		}
		return true
	}
}

// PathPartial returns a path filter which skips steps in the path if they don't
// match the StepFilter. Each StepFilter must match a step, and the final StepFilter
// must match the final step in the path.
// FIXME: Partial is the wrong name for this function
func PathPartial(steps ...StepFilter) func(gocmp.Path) bool {
	return func(path gocmp.Path) bool {
		index := 0
		for _, step := range path {
			if index >= len(steps) {
				return false
			}
			if steps[index](step) {
				index++
			}
		}
		return index == len(steps)
	}
}

// Step applies multiple StepFilter to the same step.
func Step(filters ...StepFilter) StepFilter {
	return func(step gocmp.PathStep) bool {
		for _, filter := range filters {
			if !filter(step) {
				return false
			}
		}
		return true
	}
}

// StepFilter is a function type which returns true when the step matches the filter.
type StepFilter func(gocmp.PathStep) bool

func Any(_ gocmp.PathStep) bool {
	return true
}

func Type(typ interface{}) StepFilter {
	return func(step gocmp.PathStep) bool {
		switch typ.(type) {
		case reflect.Type:
			return step.Type() == typ
		default:
			return step.Type() == reflect.TypeOf(typ)
		}
	}
}

func Slice(step gocmp.PathStep) bool {
	_, ok := step.(gocmp.SliceIndex)
	return ok
}

func Index(index int) StepFilter {
	return func(step gocmp.PathStep) bool {
		sliceIndex, ok := step.(gocmp.SliceIndex)
		if ok {
			return sliceIndex.Key() == index
		}
		structField, ok := step.(gocmp.StructField)
		return ok && structField.Index() == index
	}
}

func SplitKeys(x, y int) StepFilter {
	return func(step gocmp.PathStep) bool {
		sliceIndex, ok := step.(gocmp.SliceIndex)
		if !ok {
			return false
		}
		keyX, keyY := sliceIndex.SplitKeys()
		return keyX == x && keyY == y
	}
}

func Field(name string) StepFilter {
	return func(step gocmp.PathStep) bool {
		structField, ok := step.(gocmp.StructField)
		return ok && structField.Name() == name
	}
}

func Indirect(step gocmp.PathStep) bool {
	_, ok := step.(gocmp.Indirect)
	return ok
}

func TypeAssertion(step gocmp.PathStep) bool {
	_, ok := step.(gocmp.TypeAssertion)
	return ok
}

func Map(step gocmp.PathStep) bool {
	_, ok := step.(gocmp.MapIndex)
	return ok
}

func MapKey(key interface{}) StepFilter {
	return func(step gocmp.PathStep) bool {
		mapIndex, ok := step.(gocmp.MapIndex)
		return ok && key == mapIndex.Key().Interface()
	}
}
