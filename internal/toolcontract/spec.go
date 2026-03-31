package toolcontract

import "strings"

type Spec struct {
	Name              string
	Purpose           string
	Description       string
	InputContract     string
	OutputContract    string
	Usage             string
	InputJSONExample  string
	OutputJSONExample string
}

func (s Spec) Normalized() Spec {
	s.Name = strings.TrimSpace(s.Name)
	s.Purpose = strings.TrimSpace(s.Purpose)
	s.Description = strings.TrimSpace(s.Description)
	s.InputContract = strings.TrimSpace(s.InputContract)
	s.OutputContract = strings.TrimSpace(s.OutputContract)
	s.Usage = strings.TrimSpace(s.Usage)
	s.InputJSONExample = strings.TrimSpace(s.InputJSONExample)
	s.OutputJSONExample = strings.TrimSpace(s.OutputJSONExample)
	if s.Description == "" {
		s.Description = s.Purpose
	}
	if s.Purpose == "" {
		s.Purpose = s.Description
	}
	return s
}
