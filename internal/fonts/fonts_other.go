//go:build !linux && !darwin && !windows

package fonts

func systemCandidates() []Candidate {
	return nil
}
