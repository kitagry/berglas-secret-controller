package controllers

import (
	batchv1alpha1 "github.com/kitagry/berglas-secret-controller/api/v1alpha1"
)

func setCondition(status *batchv1alpha1.BerglasSecretStatus, newCondition batchv1alpha1.BerglasSecretCondition) {
	if status.Conditions == nil {
		status.Conditions = make([]batchv1alpha1.BerglasSecretCondition, 0, 1)
	}

	if len(status.Conditions) > 0 {
		// Don't add duplicate conditions
		lastCondition := status.Conditions[len(status.Conditions)-1]
		if lastCondition.Status == newCondition.Status && lastCondition.Reason == newCondition.Reason {
			return
		}
	}

	newConditions := filterOutCondition(status.Conditions, newCondition.Type)
	status.Conditions = append(newConditions, newCondition)
}

func filterOutCondition(conditions []batchv1alpha1.BerglasSecretCondition, conditionType batchv1alpha1.BerglasSecretConditionType) []batchv1alpha1.BerglasSecretCondition {
	newConditions := make([]batchv1alpha1.BerglasSecretCondition, 0)
	for _, c := range conditions {
		if c.Type != conditionType {
			newConditions = append(newConditions, c)
		}
	}
	return newConditions
}
