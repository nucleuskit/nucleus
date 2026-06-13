package config

import (
	"context"
	"fmt"
)

type Values map[string]any

type Update struct {
	Values   Values
	Source   string
	Revision string
}

type Source struct {
	Name     string
	Kind     string
	Location string
	Priority int
}

type Loader interface {
	Load(ctx context.Context) (Values, error)
}

type Watcher interface {
	Watch(ctx context.Context) (<-chan Update, error)
}

type Scanner interface {
	Scan(ctx context.Context, target any) error
}

type Provider interface {
	Loader
	Watcher
	Scanner
	Sources() []Source
}

const maxCloneDepth = 100

func CloneValues(values Values) Values {
	if values == nil {
		return Values{}
	}

	result, err := cloneValueWithDepthLimit(values, 0)
	if err != nil {
		panic(err)
	}

	if v, ok := result.(Values); ok {
		return v
	}

	panic("unexpected type after cloning")
}

func cloneValueWithDepthLimit(value any, depth int) (any, error) {
	if depth > maxCloneDepth {
		return nil, fmt.Errorf("configuration nesting depth exceeds maximum allowed depth of %d", maxCloneDepth)
	}

	switch typed := value.(type) {
	case Values:
		return cloneMap(typed, depth, func(m map[string]any, k string, v any) {
			m[k] = v
		})
	case map[string]any:
		return cloneMap(typed, depth, func(m map[string]any, k string, v any) {
			m[k] = v
		})
	case []any:
		return cloneSlice(typed, depth)
	default:
		return typed, nil
	}
}

func cloneMap(src map[string]any, depth int, setter func(map[string]any, string, any)) (any, error) {
	cloned := make(map[string]any, len(src))
	for key, val := range src {
		child, err := cloneValueWithDepthLimit(val, depth+1)
		if err != nil {
			return nil, err
		}
		setter(cloned, key, child)
	}
	return cloned, nil
}

func cloneSlice(src []any, depth int) (any, error) {
	cloned := make([]any, len(src))
	for i, val := range src {
		child, err := cloneValueWithDepthLimit(val, depth+1)
		if err != nil {
			return nil, err
		}
		cloned[i] = child
	}
	return cloned, nil
}
