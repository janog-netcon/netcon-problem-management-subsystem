package printers

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/janog-netcon/netcon-problem-management-subsystem/api/v1alpha1"
	"github.com/janog-netcon/netcon-problem-management-subsystem/pkg/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/duration"
)

func generateTableBaseForProblemEnvironment() *metav1.Table {
	return &metav1.Table{
		ColumnDefinitions: []metav1.TableColumnDefinition{
			{Name: "Name", Type: "string", Format: "name"},
			{Name: "Ready", Type: "string"},
			{Name: "Status", Type: "string"},
			{Name: "Age", Type: "string"},
			{Name: "Worker", Type: "string", Priority: 1},
			{Name: "Containers", Type: "string", Priority: 1},
			{Name: "Password", Type: "string", Priority: 1},
		},
	}
}

func getReadyForProblemEnvironment(problemEnvironment *v1alpha1.ProblemEnvironment) string {
	var total, ready uint
	containerStatuses := problemEnvironment.Status.Containers.Details
	for _, containerStatus := range containerStatuses {
		total += 1
		if containerStatus.Ready {
			ready += 1
		}
	}
	return fmt.Sprintf("%d/%d", ready, total)
}

func getStatusForProblemEnvironment(problemEnvironment *v1alpha1.ProblemEnvironment) string {
	scheduled := util.GetProblemEnvironmentCondition(
		problemEnvironment,
		v1alpha1.ProblemEnvironmentConditionScheduled,
	) == metav1.ConditionTrue

	deployed := util.GetProblemEnvironmentCondition(
		problemEnvironment,
		v1alpha1.ProblemEnvironmentConditionDeployed,
	) == metav1.ConditionTrue

	assigned := util.GetProblemEnvironmentCondition(
		problemEnvironment,
		v1alpha1.ProblemEnvironmentConditionAssigned,
	) == metav1.ConditionTrue

	if !scheduled {
		return "Scheduling"
	}

	if !deployed {
		return "deploying"
	}

	if !assigned {
		return "Unassigned"
	}

	return "Assigned"
}

func getContainersForProblemEnvironment(problemEnvironment *v1alpha1.ProblemEnvironment) string {
	containerNames := []string{}
	containerStatuses := problemEnvironment.Status.Containers.Details
	for _, containerStatus := range containerStatuses {
		containerNames = append(containerNames, containerStatus.Name)
	}
	sort.Strings(containerNames)
	return strings.Join(containerNames, ",")
}

func translateTimestampSince(timestamp metav1.Time) string {
	if timestamp.IsZero() {
		return "<unknown>"
	}

	return duration.HumanDuration(time.Since(timestamp.Time))
}

func generateTableRowForProblemEnvironment(
	problemEnvironment *v1alpha1.ProblemEnvironment,
	options GenerateOptions,
) metav1.TableRow {
	name := problemEnvironment.Name
	ready := getReadyForProblemEnvironment(problemEnvironment)
	status := getStatusForProblemEnvironment(problemEnvironment)
	age := translateTimestampSince(problemEnvironment.CreationTimestamp)
	password := problemEnvironment.Status.Password
	worker := problemEnvironment.Spec.WorkerName
	containers := getContainersForProblemEnvironment(problemEnvironment)

	cells := []interface{}{name, ready, status, age}
	if options.Wide {
		cells = append(cells, worker, containers, password)
	}

	return metav1.TableRow{Cells: cells}
}

func generateTableForProblemEnvironment(
	problemEnvironment *v1alpha1.ProblemEnvironment,
	options GenerateOptions,
) (*metav1.Table, error) {
	table := generateTableBaseForProblemEnvironment()
	table.ResourceVersion = problemEnvironment.ResourceVersion

	row := generateTableRowForProblemEnvironment(problemEnvironment, options)
	table.Rows = []metav1.TableRow{row}

	return table, nil
}

func generateTableForProblemEnvironmentList(
	problemEnvironments *v1alpha1.ProblemEnvironmentList,
	options GenerateOptions,
) (*metav1.Table, error) {
	table := generateTableBaseForProblemEnvironment()
	table.ResourceVersion = problemEnvironments.ResourceVersion

	for _, problemEnvironment := range problemEnvironments.Items {
		row := generateTableRowForProblemEnvironment(&problemEnvironment, options)
		table.Rows = append(table.Rows, row)
	}

	return table, nil
}
