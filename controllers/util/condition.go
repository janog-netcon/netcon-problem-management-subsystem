package util

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	netconv1alpha1 "github.com/janog-netcon/netcon-problem-management-subsystem/api/v1alpha1"
)

func SetProblemEnvironmentCondition(
	problemEnvironment *netconv1alpha1.ProblemEnvironment,
	conditionType netconv1alpha1.ProblemEnvironmentConditionType,
	status metav1.ConditionStatus,
	reason, message string,
) {
	now := metav1.NewTime(time.Now())

	for i := range problemEnvironment.Status.Conditions {
		condition := &problemEnvironment.Status.Conditions[i]
		if condition.Type != string(conditionType) {
			continue
		}

		if condition.Status != status {
			condition.Status = status
			condition.LastTransitionTime = now
		}

		condition.ObservedGeneration = problemEnvironment.ObjectMeta.Generation
		condition.Reason = reason
		condition.Message = message

		return
	}

	conditions := append(problemEnvironment.Status.Conditions, metav1.Condition{
		Type:               string(conditionType),
		Status:             status,
		ObservedGeneration: problemEnvironment.ObjectMeta.Generation,
		LastTransitionTime: now,
		Reason:             reason,
		Message:            message,
	})

	problemEnvironment.Status.Conditions = conditions
}
