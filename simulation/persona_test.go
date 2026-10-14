package simulation

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRegistry(t *testing.T) {
	r := NewRegistry()
	require.NotNil(t, r)

	personas := r.List()
	assert.Len(t, personas, 12)

	for _, p := range personas {
		assert.NotEmpty(t, p.ID, "persona ID should not be empty")
		assert.NotEmpty(t, p.Name, "persona Name should not be empty")
		assert.NotEmpty(t, p.Type, "persona Type should not be empty")
		assert.NotEmpty(t, p.RoleTitle, "persona RoleTitle should not be empty")
		assert.NotEmpty(t, p.Department, "persona Department should not be empty")
		assert.NotEmpty(t, p.Actions, "persona Actions should not be empty")

		for _, action := range p.Actions {
			assert.NotEmpty(t, action.Name, "action Name should not be empty")
			assert.NotEmpty(t, action.Description, "action Description should not be empty")
			assert.NotNil(t, action.Execute, "action Execute func should not be nil")
		}
	}
}

func TestRegistryRegister(t *testing.T) {
	r := &Registry{
		personas: make(map[string]*Persona),
		order:    make([]string, 0),
	}

	p1 := &Persona{
		ID:         "emp-test-01",
		Name:       "Test Person One",
		Type:       "human",
		RoleTitle:  "Tester",
		Department: "QA",
		Actions:    []Action{},
	}

	r.Register(p1)
	assert.Equal(t, 1, len(r.List()))

	fetched, ok := r.Get("emp-test-01")
	require.True(t, ok)
	assert.Equal(t, "Test Person One", fetched.Name)

	// Registering an updated persona with the same ID
	p1Updated := &Persona{
		ID:         "emp-test-01",
		Name:       "Test Person One Updated",
		Type:       "human",
		RoleTitle:  "Senior Tester",
		Department: "QA",
		Actions:    []Action{},
	}

	r.Register(p1Updated)
	assert.Equal(t, 1, len(r.List()), "registering existing ID should not duplicate in order list")

	fetchedUpdated, ok := r.Get("emp-test-01")
	require.True(t, ok)
	assert.Equal(t, "Test Person One Updated", fetchedUpdated.Name)
	assert.Equal(t, "Senior Tester", fetchedUpdated.RoleTitle)
}

func TestRegistryGet(t *testing.T) {
	r := &Registry{
		personas: make(map[string]*Persona),
		order:    make([]string, 0),
	}

	p1 := &Persona{
		ID:         "emp-human-alice",
		Name:       "Alice Vance",
		Type:       "human",
		RoleTitle:  "HR Head",
		Department: "Workforce",
	}
	p2 := &Persona{
		ID:         "emp-feline-garfield",
		Name:       "Garfield",
		Type:       "feline",
		RoleTitle:  "COO",
		Department: "Executive",
	}

	r.Register(p1)
	r.Register(p2)

	// Test Get by ID
	gotByID, ok := r.Get("emp-human-alice")
	require.True(t, ok)
	assert.Equal(t, p1, gotByID)

	// Test Get by Name
	gotByName, ok := r.Get("Garfield")
	require.True(t, ok)
	assert.Equal(t, p2, gotByName)

	// Test Get by non-existent ID or Name
	gotNonExistent, ok := r.Get("non-existent-persona")
	assert.False(t, ok)
	assert.Nil(t, gotNonExistent)
}

func TestRegistryList(t *testing.T) {
	r := &Registry{
		personas: make(map[string]*Persona),
		order:    make([]string, 0),
	}

	p1 := &Persona{ID: "p1", Name: "Persona 1"}
	p2 := &Persona{ID: "p2", Name: "Persona 2"}
	p3 := &Persona{ID: "p3", Name: "Persona 3"}

	r.Register(p1)
	r.Register(p2)
	r.Register(p3)

	list := r.List()
	require.Len(t, list, 3)
	assert.Equal(t, "p1", list[0].ID)
	assert.Equal(t, "p2", list[1].ID)
	assert.Equal(t, "p3", list[2].ID)
}
