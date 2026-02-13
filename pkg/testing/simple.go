package testing

type TestManager struct{}

func NewTestManager(projectDir string) *TestManager {
	return &TestManager{}
}

func (tm *TestManager) RunTests() ([]string, error) {
	return []string{"test-1-passed", "test-2-passed"}, nil
}
