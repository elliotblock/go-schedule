package database

import (
	"encoding/json"
	"html/template"
	"regexp"
	"strings"
)

// A Parent is a type that will be referenced by
// children.
// ForeignKey returns whatever the children will store
// to establish their relationship to the parent.
type Parent interface {
	ForeignKey() interface{}
}

// Structs that implement Queryer can be stored
// and retrieved from the database with Put and
// Get, respectively.
// PrimaryKey returns the unique identifier for
// the struct.
// TableName returns the name of the database table
// that should be queried.
// NOTE: The corresponding database table must
// be setup with the proper field values.
type Queryer interface {
	PrimaryKey() interface{}
	TableName() string
}

// A Department is a UW department that has many classes
type Dept struct {
	Title        string
	Abbreviation string // primary key
	Link         string
}

// PrimaryKey returns the dept's Abbreviation.
func (d Dept) PrimaryKey() interface{} {
	return d.Abbreviation
}

// TableName returns the name of the SQL table corresponding
// to Dept structs.
func (d Dept) TableName() string {
	return "depts"
}

// A Class is UW class that has many sections.
type Class struct {
	DeptAbbreviation string // foreign key
	AbbreviationCode string // primary key
	Abbreviation     string
	Code             string
	Title            string
	Description      string
	Index            int
}

// PrimaryKey returns the class's AbbreviationCode.
func (c Class) PrimaryKey() interface{} {
	return c.AbbreviationCode
}

// TableName returns the name of the SQL table corresponding
// to Class structs.
func (c Class) TableName() string {
	return "classes"
}

// Classes wraps a slice of Class structs so they
// can implement a ForeignKey method.
type Classes []Class

// DescriptionHTML outputs non-escaped HTML of a class's
// for use in a template.
func (c Class) DescriptionHTML() template.HTML {
	root := "http://www.washington.edu"
	d := c.Description
	// Add root index to links
	re := `<a href="`
	offset := len(re)
	// For each link, insert the root at the index of the match, adding offset generated by previous link insertions
	if matches := regexp.MustCompile(`<a href="`).FindAllStringIndex(strings.ToLower(d), -1); matches != nil {
		for i, match := range matches {
			perLinkOffset := i * len(root)
			d = d[:(match[0]+offset+perLinkOffset)] + root + d[(match[0]+offset+perLinkOffset):]
		}
	}
	// add opening <B> tag
	d = "<B>" + d
	return template.HTML(d)
}

// A Class is a UW class represented on the time schedule.
type Sect struct {
	ClassDeptAbbreviation string // foreign key
	Restriction           string
	SLN                   string // primary key
	Section               string
	Credit                string
	MeetingTimes          string // JSON representation, TODO (kvu787): represent as seperate struct
	Instructor            string
	Status                string
	TakenSpots            int
	TotalSpots            int
	Grades                string
	Fee                   string
	Other                 string
	Info                  string
}

// PrimaryKey returns the sect's SLN.
func (s Sect) PrimaryKey() interface{} {
	return s.SLN
}

// TableName returns the name of the SQL table corresponding
// to Sect structs.
func (s Sect) TableName() string {
	return "sects"
}

// GetMeetingTimes parses the JSON representation of meeting
// times from the Section.
// Returns a slice of MeetingTime structs, or an empty slice
// if the section has no MeetingTime's.
func (s Sect) GetMeetingTimes() ([]MeetingTime, error) {
	var meetingTimes []MeetingTime
	if err := json.Unmarshal([]byte(s.MeetingTimes), &meetingTimes); err != nil {
		return nil, err
	}
	return meetingTimes, nil
}

// IsQuizSection indicates if this Sect is a quiz section.
func (s Sect) IsQuizSection() bool {
	if s.Credit == "QZ" {
		return true
	} else {
		return false
	}
}

// IsOpen indicates if this Sect has open spots.
func (s Sect) IsOpen() bool {
	if s.TotalSpots-s.TakenSpots < 1 {
		return false
	} else {
		return true
	}
}

// IsFreshmen indicates if this Sect is restricted to freshmen
// by looking for key phrases/words in Sect.Info.
// TODO (kvu787): use regexps
func (s Sect) IsFreshmen() bool {
	keyPhrases := []string{
		// "freshmen interest grp students only",
		// "freshman interest grp students only",
		// "freshmen interest grp only",
		// "freshman interest grp only",
		// "freshmen interest group students only",
		// "freshman interest group students only",
		// "freshmen interest group only",
		// "freshman interest group only",
		// "open only to entering freshmen",
		// "open only to entering freshman",
		// "open to entering freshmen only",
		// "open to entering freshman only",
		// "opening only to entering freshmen",
		"freshmen",
		"freshman",
	}
	info := strings.Replace(
		strings.ToLower(s.Info), "\n", " ", -1)

	for _, v := range keyPhrases {
		if strings.Contains(info, v) {
			return true
		}
	}
	return false
}

// IsWithdrawal indicates if this Sect pending withdrawal by
// looking for key phrases/words in Sect.Info.
func (s Sect) IsWithdrawal() bool {
	keyPhrases := []string{"withdrawl", "withdrawal"}
	info := strings.ToLower(s.Info)
	for _, v := range keyPhrases {
		if strings.Contains(info, v) {
			return true
		}
	}
	return false
}

// GetRestriction returns a map of possible restriction
// symbols to booleans depending on if they apply to this
// Section.
func (s Sect) GetRestriction() []map[string]bool {
	allTokens := []string{"Restr", "IS", ">"}
	tokens := make([]map[string]bool, len(allTokens))
	for i, v := range allTokens {
		if strings.Contains(s.Restriction, v) {
			tokens[i] = map[string]bool{v: true}
		} else {
			tokens[i] = map[string]bool{v: false}
		}
	}
	return tokens
}

// GetGradesTokens returns a map of possible grade
// tokens to booleans depending on if they apply
// to this Section.
func (s Sect) GetGradesTokens() []map[string]bool {
	allTokens := []string{"CR/NC"}
	tokens := make([]map[string]bool, len(allTokens))
	for i, v := range allTokens {
		if strings.Contains(s.Credit, v) {
			tokens[i] = map[string]bool{v: true}
		} else {
			tokens[i] = map[string]bool{v: false}
		}
	}
	return tokens
}

// GetOtherTokens returns a map of possible other
// tokens to booleans depending on if they apply
// to this Section.
func (s Sect) GetOtherTokens() []map[string]bool {
	allTokens := []string{"D", "H", "J", "R", "S", "W", "%", "#"}
	tokens := make([]map[string]bool, len(allTokens))
	for i, v := range allTokens {
		if strings.Contains(s.Other, v) {
			tokens[i] = map[string]bool{v: true}
		} else {
			tokens[i] = map[string]bool{v: false}
		}
	}
	return tokens
}

// A MeetingTime represents when a class is held. Some Sect's
// have multiple meeting times.
// A MeetingTime belongs to the Sect with Sln 'SectSln'.
type MeetingTime struct {
	Days     string
	Time     string
	Building string
	Room     string
}

// MapDays returns a map of possible days to
// booleans depending on what days this Section
// is held.
func (m MeetingTime) MapDays() map[string]bool {
	days := strings.ToLower(m.Days)
	dayMap := make(map[string]bool)
	daysSlice := []string{"m", "w", "f", "th", "t"}
	for _, day := range daysSlice {
		if strings.Contains(days, day) {
			dayMap[day] = true
			days = strings.Replace(days, day, "", -1)
		}
	}
	return dayMap
}
