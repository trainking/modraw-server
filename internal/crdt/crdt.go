package crdt

import (
	"encoding/json"
)

const (
	OpElementAdd     = "element_add"
	OpElementUpdate  = "element_update"
	OpElementDelete  = "element_delete"
	OpElementsReorder = "elements_reorder"
	OpCanvasUpdate   = "canvas_update"
)

type Operation struct {
	Type    string          `json:"op"`
	ElemID  string          `json:"elem_id,omitempty"`
	Version int64           `json:"version"`
	Props   json.RawMessage `json:"props,omitempty"`
	ElemIDs []string        `json:"elem_ids,omitempty"`
}

type Element struct {
	ID      string          `json:"id"`
	Version int64           `json:"_v"`
	Data    json.RawMessage `json:"data"`
}

type CanvasState struct {
	Elements []Element       `json:"elements"`
	Props    json.RawMessage `json:"props"`
	Version  int64           `json:"_v"`
}

func ParseState(data json.RawMessage) (*CanvasState, error) {
	state := &CanvasState{Elements: make([]Element, 0)}
	if len(data) == 0 || string(data) == "{}" {
		return state, nil
	}
	if err := json.Unmarshal(data, state); err != nil {
		return nil, err
	}
	if state.Elements == nil {
		state.Elements = make([]Element, 0)
	}
	return state, nil
}

func (s *CanvasState) Serialize() json.RawMessage {
	b, _ := json.Marshal(s)
	return json.RawMessage(b)
}

func (s *CanvasState) Apply(op Operation) bool {
	switch op.Type {
	case OpElementAdd:
		return s.applyElementAdd(op)
	case OpElementUpdate:
		return s.applyElementUpdate(op)
	case OpElementDelete:
		return s.applyElementDelete(op)
	case OpElementsReorder:
		return s.applyElementsReorder(op)
	case OpCanvasUpdate:
		return s.applyCanvasUpdate(op)
	}
	return false
}

func (s *CanvasState) applyElementAdd(op Operation) bool {
	for _, e := range s.Elements {
		if e.ID == op.ElemID {
			return false // already exists
		}
	}
	s.Elements = append(s.Elements, Element{
		ID:      op.ElemID,
		Version: op.Version,
		Data:    op.Props,
	})
	s.Version++
	return true
}

func (s *CanvasState) applyElementUpdate(op Operation) bool {
	for i, e := range s.Elements {
		if e.ID == op.ElemID {
			if op.Version <= e.Version {
				return false // stale or duplicate
			}
			merged := mergeJSON(e.Data, op.Props)
			s.Elements[i].Data = merged
			s.Elements[i].Version = op.Version
			s.Version++
			return true
		}
	}
	// Element not found — add it (handles out-of-order delivery where add arrives after update)
	s.Elements = append(s.Elements, Element{
		ID:      op.ElemID,
		Version: op.Version,
		Data:    op.Props,
	})
	s.Version++
	return true
}

func (s *CanvasState) applyElementDelete(op Operation) bool {
	for i, e := range s.Elements {
		if e.ID == op.ElemID {
			s.Elements = append(s.Elements[:i], s.Elements[i+1:]...)
			s.Version++
			return true
		}
	}
	return false // not found, already deleted
}

func (s *CanvasState) applyElementsReorder(op Operation) bool {
	if len(op.ElemIDs) == 0 {
		return false
	}
	byID := make(map[string]Element, len(s.Elements))
	for _, e := range s.Elements {
		byID[e.ID] = e
	}
	reordered := make([]Element, 0, len(s.Elements))
	for _, id := range op.ElemIDs {
		if e, ok := byID[id]; ok {
			reordered = append(reordered, e)
			delete(byID, id)
		}
	}
	// Append any remaining elements not in the reorder list
	for _, e := range s.Elements {
		if _, ok := byID[e.ID]; ok {
			reordered = append(reordered, e)
		}
	}
	s.Elements = reordered
	s.Version++
	return true
}

func (s *CanvasState) applyCanvasUpdate(op Operation) bool {
	s.Props = mergeJSON(s.Props, op.Props)
	s.Version++
	return true
}

func mergeJSON(base, patch json.RawMessage) json.RawMessage {
	if len(base) == 0 {
		return patch
	}
	if len(patch) == 0 {
		return base
	}

	var baseMap map[string]json.RawMessage
	var patchMap map[string]json.RawMessage
	if err := json.Unmarshal(base, &baseMap); err != nil {
		return patch
	}
	if err := json.Unmarshal(patch, &patchMap); err != nil {
		return base
	}
	if baseMap == nil {
		baseMap = make(map[string]json.RawMessage)
	}

	for k, v := range patchMap {
		baseMap[k] = v
	}

	result, err := json.Marshal(baseMap)
	if err != nil {
		return base
	}
	return json.RawMessage(result)
}
