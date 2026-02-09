// Package inttest provides an integration test framework for prompt-pulse v2.
// It validates cross-package interactions between config, layout, theme,
// preset, banner, shell, starship, tui, cache, and component packages.
//
// The framework supports test suites with setup/teardown, tagged test
// filtering, and pipeline-based stage execution for verifying end-to-end
// workflows.
package inttest

import (
	"testing"
	"time"
)

// TestSuite groups related integration tests with optional setup and teardown.
type TestSuite struct {
	// Name is the display name for this suite.
	Name string

	// Tests is the ordered list of integration tests in this suite.
	Tests []IntegrationTest

	// Setup runs once before any test in the suite. If it returns an error,
	// all tests in the suite are skipped.
	Setup func() error

	// Teardown runs once after all tests complete, regardless of pass/fail.
	Teardown func() error
}

// IntegrationTest defines a single integration test with metadata.
type IntegrationTest struct {
	// Name identifies this test.
	Name string

	// Run is the test function.
	Run func(t *testing.T)

	// Tags categorize this test for filtered execution.
	Tags []string

	// Timeout is the maximum duration for this test. Zero means no timeout.
	Timeout time.Duration
}

// TestResult captures the outcome of a single integration test execution.
type TestResult struct {
	// Name is the test name.
	Name string

	// Passed is true if the test completed without failures.
	Passed bool

	// Duration is how long the test took to execute.
	Duration time.Duration

	// Error holds the error message if the test failed.
	Error string

	// Output holds any captured test output.
	Output string
}

// NewSuite creates a new empty TestSuite with the given name.
func NewSuite(name string) *TestSuite {
	return &TestSuite{
		Name: name,
	}
}

// Add appends a new test to the suite with the given name, function, and
// optional tags.
func (s *TestSuite) Add(name string, fn func(t *testing.T), tags ...string) {
	s.Tests = append(s.Tests, IntegrationTest{
		Name: name,
		Run:  fn,
		Tags: tags,
	})
}

// Run executes all tests in the suite as subtests of t. Setup is called
// before the first test and Teardown after the last.
func (s *TestSuite) Run(t *testing.T) {
	t.Helper()

	if s.Setup != nil {
		if err := s.Setup(); err != nil {
			t.Fatalf("suite %q setup failed: %v", s.Name, err)
		}
	}

	if s.Teardown != nil {
		t.Cleanup(func() {
			if err := s.Teardown(); err != nil {
				t.Errorf("suite %q teardown failed: %v", s.Name, err)
			}
		})
	}

	for _, test := range s.Tests {
		test := test // capture
		t.Run(test.Name, func(t *testing.T) {
			if test.Timeout > 0 {
				timer := time.AfterFunc(test.Timeout, func() {
					t.Errorf("test %q exceeded timeout of %v", test.Name, test.Timeout)
				})
				defer timer.Stop()
			}
			test.Run(t)
		})
	}
}

// RunTagged executes only the tests whose Tags include at least one of
// the specified tags. If no tags are provided, no tests are executed.
func (s *TestSuite) RunTagged(t *testing.T, tags ...string) {
	t.Helper()

	if len(tags) == 0 {
		return
	}

	tagSet := make(map[string]bool, len(tags))
	for _, tag := range tags {
		tagSet[tag] = true
	}

	if s.Setup != nil {
		if err := s.Setup(); err != nil {
			t.Fatalf("suite %q setup failed: %v", s.Name, err)
		}
	}

	if s.Teardown != nil {
		t.Cleanup(func() {
			if err := s.Teardown(); err != nil {
				t.Errorf("suite %q teardown failed: %v", s.Name, err)
			}
		})
	}

	for _, test := range s.Tests {
		if !itHasMatchingTag(test.Tags, tagSet) {
			continue
		}
		test := test // capture
		t.Run(test.Name, func(t *testing.T) {
			if test.Timeout > 0 {
				timer := time.AfterFunc(test.Timeout, func() {
					t.Errorf("test %q exceeded timeout of %v", test.Name, test.Timeout)
				})
				defer timer.Stop()
			}
			test.Run(t)
		})
	}
}

// itHasMatchingTag returns true if any of testTags exists in the tagSet.
func itHasMatchingTag(testTags []string, tagSet map[string]bool) bool {
	for _, tag := range testTags {
		if tagSet[tag] {
			return true
		}
	}
	return false
}
