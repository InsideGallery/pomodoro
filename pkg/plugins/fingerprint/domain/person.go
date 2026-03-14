package domain

// Person represents a suspect in the fingerprint database.
type Person struct {
	Name          string
	AvatarKey     string // resource key for avatar image
	FingerprintID string // UniqueID of their fingerprint
}

// Database is an in-memory person lookup by fingerprint ID.
type Database struct {
	persons map[string]*Person // fingerprintID → person
	all     []*Person
}

// NewDatabase creates an empty person database.
func NewDatabase() *Database {
	return &Database{
		persons: make(map[string]*Person),
	}
}

// Add registers a person in the database.
func (db *Database) Add(p *Person) {
	db.persons[p.FingerprintID] = p
	db.all = append(db.all, p)
}

// Lookup finds a person by fingerprint ID. Returns nil if not found.
func (db *Database) Lookup(fingerprintID string) *Person {
	return db.persons[fingerprintID]
}

// All returns all persons in the database.
func (db *Database) All() []*Person {
	return db.all
}

// Size returns the number of persons in the database.
func (db *Database) Size() int {
	return len(db.all)
}
