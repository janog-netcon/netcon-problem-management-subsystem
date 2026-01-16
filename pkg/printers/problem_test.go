package printers

import (
	"bytes"
	"strings"
	"testing"

	"github.com/janog-netcon/netcon-problem-management-subsystem/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// createTestProblemEnvironmentList creates a sample ProblemEnvironmentList for testing
func createTestProblemEnvironmentList() *v1alpha1.ProblemEnvironmentList {
	return &v1alpha1.ProblemEnvironmentList{
		Items: []v1alpha1.ProblemEnvironment{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "pe1",
				},
				Spec: v1alpha1.ProblemEnvironmentSpec{
					WorkerName: "worker1",
				},
				Status: v1alpha1.ProblemEnvironmentStatus{
					Conditions: []metav1.Condition{
						{
							Type:   string(v1alpha1.ProblemEnvironmentConditionScheduled),
							Status: metav1.ConditionTrue,
						},
						{
							Type:   string(v1alpha1.ProblemEnvironmentConditionDeployed),
							Status: metav1.ConditionTrue,
						},
						{
							Type:   string(v1alpha1.ProblemEnvironmentConditionReady),
							Status: metav1.ConditionTrue,
						},
						{
							Type:   string(v1alpha1.ProblemEnvironmentConditionAssigned),
							Status: metav1.ConditionFalse,
						},
					},
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "pe2",
				},
				Spec: v1alpha1.ProblemEnvironmentSpec{
					WorkerName: "worker2",
				},
				Status: v1alpha1.ProblemEnvironmentStatus{
					Conditions: []metav1.Condition{
						{
							Type:   string(v1alpha1.ProblemEnvironmentConditionScheduled),
							Status: metav1.ConditionTrue,
						},
						{
							Type:   string(v1alpha1.ProblemEnvironmentConditionDeployed),
							Status: metav1.ConditionTrue,
						},
						{
							Type:   string(v1alpha1.ProblemEnvironmentConditionReady),
							Status: metav1.ConditionTrue,
						},
						{
							Type:   string(v1alpha1.ProblemEnvironmentConditionAssigned),
							Status: metav1.ConditionTrue,
						},
					},
				},
			},
		},
	}
}

func TestGenerateTableForProblemEnvironmentList(t *testing.T) {
	peList := createTestProblemEnvironmentList()

	// Generate table
	table, err := generateTableForProblemEnvironmentList(peList, GenerateOptions{})
	if err != nil {
		t.Fatalf("Error generating table: %v", err)
	}

	// Verify that we have 2 rows (one for each PE)
	if len(table.Rows) != 2 {
		t.Errorf("Expected 2 rows, got %d", len(table.Rows))
	}

	// Verify the rows have the expected names
	if len(table.Rows) > 0 {
		if table.Rows[0].Cells[0] != "pe1" {
			t.Errorf("Expected first row name to be 'pe1', got '%v'", table.Rows[0].Cells[0])
		}
	}
	if len(table.Rows) > 1 {
		if table.Rows[1].Cells[0] != "pe2" {
			t.Errorf("Expected second row name to be 'pe2', got '%v'", table.Rows[1].Cells[0])
		}
	}
}

func TestPrintProblemEnvironmentList(t *testing.T) {
	peList := createTestProblemEnvironmentList()

	// Print the list
	var buf bytes.Buffer
	err := PrintObject(&buf, peList, PrintOptions{})
	if err != nil {
		t.Fatalf("Error printing object: %v", err)
	}

	output := buf.String()
	
	// Check that both PE names appear in the output
	if !strings.Contains(output, "pe1") {
		t.Errorf("Expected output to contain 'pe1', got:\n%s", output)
	}
	if !strings.Contains(output, "pe2") {
		t.Errorf("Expected output to contain 'pe2', got:\n%s", output)
	}
	
	// Check that the header is present
	if !strings.Contains(output, "NAME") {
		t.Errorf("Expected output to contain header 'NAME', got:\n%s", output)
	}
}
