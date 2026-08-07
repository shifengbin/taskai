//go:build darwin && !cgo

package fonts

func systemCandidates() []Candidate {
	return nil
}
