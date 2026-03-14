package domain

// CaseStatus represents the state of a fingerprint case.
type CaseStatus int

const (
	CaseUnsolved CaseStatus = iota
	CaseSolved
	CaseFailed
)

// Case represents a forensic case with an unknown fingerprint to identify.
type Case struct {
	ID     int
	Status CaseStatus
	Puzzle *Fingerprint // the puzzle fingerprint (partially filled)

	// Result (set after submission)
	SubmittedID string  // the fingerprint ID the player constructed
	MatchPerson *Person // nil if no match found
}

// NewCase creates an unsolved case with the given puzzle.
func NewCase(id int, puzzle *Fingerprint) *Case {
	return &Case{
		ID:     id,
		Status: CaseUnsolved,
		Puzzle: puzzle,
	}
}

// Submit attempts to identify the fingerprint.
// Returns the matched person (nil if unknown) and whether it's correct.
func (c *Case) Submit(db *Database, expectedID string) (*Person, bool) {
	constructedID := c.Puzzle.UniqueID()
	c.SubmittedID = constructedID

	person := db.Lookup(constructedID)
	c.MatchPerson = person

	if constructedID == expectedID && person != nil {
		c.Status = CaseSolved

		return person, true
	}

	c.Status = CaseFailed

	return person, false
}

// IsSolved returns true if the case was solved correctly.
func (c *Case) IsSolved() bool {
	return c.Status == CaseSolved
}
