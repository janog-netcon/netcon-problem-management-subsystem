package printers

import (
	"github.com/janog-netcon/netcon-problem-management-subsystem/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func generateTableBaseForWorker() *metav1.Table {
	return &metav1.Table{
		ColumnDefinitions: []metav1.TableColumnDefinition{
			{Name: "Name", Type: "string", Format: "name"},
			{Name: "Enabled", Type: "string"},
			{Name: "Age", Type: "string"},
			{Name: "CPU", Type: "string", Priority: 1},
			{Name: "Memory", Type: "string", Priority: 1},
			{Name: "IPAddress", Type: "string", Priority: 1},
			{Name: "Port", Type: "number", Priority: 1},
		},
	}
}

func generateTableRowForWorker(
	worker *v1alpha1.Worker,
	options GenerateOptions,
) metav1.TableRow {
	name := worker.Name
	enabled := !worker.Spec.DisableSchedule
	age := translateTimestampSince(worker.CreationTimestamp)
	cpu := worker.Status.WorkerInfo.CPUUsedPercent
	memory := worker.Status.WorkerInfo.MemoryUsedPercent
	ipAddress := worker.Status.WorkerInfo.ExternalIPAddress
	port := worker.Status.WorkerInfo.ExternalPort

	cells := []interface{}{name, enabled, age}
	if options.Wide {
		cells = append(cells, cpu, memory, ipAddress, port)
	}

	return metav1.TableRow{Cells: cells}
}

func generateTableForWorker(
	worker *v1alpha1.Worker,
	options GenerateOptions,
) (*metav1.Table, error) {
	table := generateTableBaseForWorker()
	table.ResourceVersion = worker.ResourceVersion

	row := generateTableRowForWorker(worker, options)
	table.Rows = []metav1.TableRow{row}

	return table, nil
}

func generateTableForWorkerList(
	workers *v1alpha1.WorkerList,
	options GenerateOptions,
) (*metav1.Table, error) {
	table := generateTableBaseForWorker()
	table.ResourceVersion = workers.ResourceVersion

	for _, worker := range workers.Items {
		row := generateTableRowForWorker(&worker, options)
		table.Rows = append(table.Rows, row)
	}

	return table, nil
}
