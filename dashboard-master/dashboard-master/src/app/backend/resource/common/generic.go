package common

import (
	"github.com/kubernetes/dashboard/src/app/backend/resource/dataselect"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GetPaginatedList is a generic function that can be used to list any type of Kubernetes resource.
func GetPaginatedList[T client.Object, U client.ObjectList](
	client kubernetes.Interface,
	namespace string,
	dsQuery *dataselect.DataSelectQuery,
	listFunc func(opts v1.ListOptions) (U, error),
) ([]T, error) {
	// Create a new list of the specified type.
	var resourceList U

	// Get the list of resources from the API server.
	resourceList, err := listFunc(v1.ListOptions{})
	if err != nil {
		return nil, err
	}

	// Create a new list of DataCell objects.
	// This is a placeholder for the actual implementation.
	// The actual implementation will convert the resource list to a list of DataCell objects.
	cells := toCells(resourceList)

	// Perform data selection and pagination.
	// This is a placeholder for the actual implementation.
	// The actual implementation will use the dataselect package to perform data selection and pagination.
	paginatedList, _ := dataselect.GenericDataSelectWithFilter(cells, dsQuery)

	return fromCells[T](paginatedList), nil
}

// toCells is a placeholder for the actual implementation.
// The actual implementation will convert the resource list to a list of DataCell objects.
func toCells[T client.ObjectList](list T) []dataselect.DataCell[client.Object] {
	return nil
}

// fromCells is a placeholder for the actual implementation.
// The actual implementation will convert a list of DataCell objects to a list of resources.
func fromCells[T client.Object](cells []dataselect.DataCell[client.Object]) []T {
	return nil
}
