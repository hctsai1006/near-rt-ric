// Copyright 2017 The Kubernetes Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package dataselect

import (
	"log"
	"sort"

	"github.com/kubernetes/dashboard/src/app/backend/errors"
	metricapi "github.com/kubernetes/dashboard/src/app/backend/integration/metric/api"
)

// DataCell is a generic interface for data cells that can be selected and sorted.
type DataCell[T any] interface {
	GetProperty(PropertyName) ComparableValue
}

// MetricDataCell extends interface of DataCells and additionally supports metric download.
type MetricDataCell[T any] interface {
	DataCell[T]
	// GetResourceSelector returns ResourceSelector for this resource. The ResourceSelector can be used to get,
	// HeapsterSelector which in turn can be used to download metrics.
	GetResourceSelector() *metricapi.ResourceSelector
}

// ComparableValue hold any value that can be compared to its own kind.
type ComparableValue interface {
	// Compares self with other value. Returns 1 if other value is smaller, 0 if they are the same, -1 if other is larger.
	Compare(ComparableValue) int
	// Returns true if self value contains or is equal to other value, false otherwise.
	Contains(ComparableValue) bool
}

// DataSelector is a generic struct for selecting, sorting, and paginating data.
type DataSelector[T any] struct {
	// GenericDataList hold generic data cells that are being selected.
	GenericDataList []DataCell[T]
	// DataSelectQuery holds instructions for data select.
	DataSelectQuery *DataSelectQuery
	// CachedResources stores resources that may be needed during data selection process
	CachedResources *metricapi.CachedResources
	// CumulativeMetricsPromises is a list of promises holding aggregated metrics for resources in GenericDataList.
	// The metrics will be calculated after calling GetCumulativeMetrics method.
	CumulativeMetricsPromises metricapi.MetricPromises
	// MetricsPromises is a list of promises holding metrics for resources in GenericDataList.
	// The metrics will be calculated after calling GetMetrics method. Metric will not be
	// aggregated and can are used to display sparklines on pod list.
	MetricsPromises metricapi.MetricPromises
}

// Implementation of sort.Interface so that we can use built-in sort function (sort.Sort) for sorting SelectableData

// Len returns the length of data inside SelectableData.
func (self DataSelector[T]) Len() int { return len(self.GenericDataList) }

// Swap swaps 2 indices inside SelectableData.
func (self DataSelector[T]) Swap(i, j int) {
	self.GenericDataList[i], self.GenericDataList[j] = self.GenericDataList[j], self.GenericDataList[i]
}

// Less compares 2 indices inside SelectableData and returns true if first index is larger.
func (self DataSelector[T]) Less(i, j int) bool {
	for _, sortBy := range self.DataSelectQuery.SortQuery.SortByList {
		a := self.GenericDataList[i].GetProperty(sortBy.Property)
		b := self.GenericDataList[j].GetProperty(sortBy.Property)
		// ignore sort completely if property name not found
		if a == nil || b == nil {
			break
		}
		cmp := a.Compare(b)
		if cmp == 0 { // values are the same. Just continue to next sortBy
			continue
		} else { // values different
			return (cmp == -1 && sortBy.Ascending) || (cmp == 1 && !sortBy.Ascending)
		}
	}
	return false
}

// Sort sorts the data inside as instructed by DataSelectQuery and returns itself to allow method chaining.
func (self *DataSelector[T]) Sort() *DataSelector[T] {
	sort.Sort(self)
	return self
}

// Filter the data inside as instructed by DataSelectQuery and returns itself to allow method chaining.
func (self *DataSelector[T]) Filter() *DataSelector[T] {
	filteredList := []DataCell[T]{}

	for _, c := range self.GenericDataList {
		matches := true
		for _, filterBy := range self.DataSelectQuery.FilterQuery.FilterByList {
			v := c.GetProperty(filterBy.Property)
			if v == nil || !v.Contains(filterBy.Value) {
				matches = false
				break
			}
		}
		if matches {
			filteredList = append(filteredList, c)
		}
	}

	self.GenericDataList = filteredList
	return self
}

func (self *DataSelector[T]) getMetrics(metricClient metricapi.MetricClient) (
	[]metricapi.MetricPromises, error) {
	metricPromises := make([]metricapi.MetricPromises, 0)

	if metricClient == nil {
		return metricPromises, errors.NewInternal("No metric client provided. Skipping metrics.")
	}

	metricNames := self.DataSelectQuery.MetricQuery.MetricNames
	if metricNames == nil {
		return metricPromises, errors.NewInternal("No metrics specified. Skipping metrics.")
	}

	selectors := make([]metricapi.ResourceSelector, len(self.GenericDataList))
	for i, dataCell := range self.GenericDataList {
		// make sure data cells support metrics
		metricDataCell, ok := dataCell.(MetricDataCell[T])
		if !ok {
			log.Printf("Data cell does not implement MetricDataCell. Skipping. %v", dataCell)
			continue
		}

		selectors[i] = *metricDataCell.GetResourceSelector()
	}

	for _, metricName := range metricNames {
		promises := metricClient.DownloadMetric(selectors, metricName, self.CachedResources)
		metricPromises = append(metricPromises, promises)
	}

	return metricPromises, nil
}

// GetMetrics downloads metrics for data cells currently present in self.GenericDataList as instructed
// by MetricQuery and inserts resulting MetricPromises to self.MetricsPromises.
func (self *DataSelector[T]) GetMetrics(metricClient metricapi.MetricClient) *DataSelector[T] {
	metricPromisesList, err := self.getMetrics(metricClient)
	if err != nil {
		log.Print(err)
		return self
	}

	metricPromises := make(metricapi.MetricPromises, 0)
	for _, promises := range metricPromisesList {
		metricPromises = append(metricPromises, promises...)
	}

	self.MetricsPromises = metricPromises
	return self
}

// GetCumulativeMetrics downloads and aggregates metrics for data cells currently present in self.GenericDataList as instructed
// by MetricQuery and inserts resulting MetricPromises to self.CumulativeMetricsPromises.
func (self *DataSelector[T]) GetCumulativeMetrics(metricClient metricapi.MetricClient) *DataSelector[T] {
	metricPromisesList, err := self.getMetrics(metricClient)
	if err != nil {
		log.Print(err)
		return self
	}

	metricNames := self.DataSelectQuery.MetricQuery.MetricNames
	if metricNames == nil {
		log.Print("No metrics specified. Skipping metrics.")
		return self
	}

	aggregations := self.DataSelectQuery.MetricQuery.Aggregations
	if aggregations == nil {
		aggregations = metricapi.OnlyDefaultAggregation
	}

	metricPromises := make(metricapi.MetricPromises, 0)
	for i, metricName := range metricNames {
		promises := metricClient.AggregateMetrics(metricPromisesList[i], metricName, aggregations)
		metricPromises = append(metricPromises, promises...)
	}

	self.CumulativeMetricsPromises = metricPromises
	return self
}

// Paginates the data inside as instructed by DataSelectQuery and returns itself to allow method chaining.
func (self *DataSelector[T]) Paginate() *DataSelector[T] {
	pQuery := self.DataSelectQuery.PaginationQuery
	dataList := self.GenericDataList
	startIndex, endIndex := pQuery.GetPaginationSettings(len(dataList))

	// Return all items if provided settings do not meet requirements
	if !pQuery.IsValidPagination() {
		return self
	}
	// Return no items if requested page does not exist
	if !pQuery.IsPageAvailable(len(self.GenericDataList), startIndex) {
		self.GenericDataList = []DataCell[T]{}
		return self
	}

	self.GenericDataList = dataList[startIndex:endIndex]
	return self
}

// GenericDataSelect takes a list of GenericDataCells and DataSelectQuery and returns selected data as instructed by dsQuery.
func GenericDataSelect[T any](dataList []DataCell[T], dsQuery *DataSelectQuery) []DataCell[T] {
	SelectableData := DataSelector[T]{
		GenericDataList: dataList,
		DataSelectQuery: dsQuery,
	}
	return SelectableData.Sort().Paginate().GenericDataList
}

// GenericDataSelectWithFilter takes a list of GenericDataCells and DataSelectQuery and returns selected data as instructed by dsQuery.
func GenericDataSelectWithFilter[T any](dataList []DataCell[T], dsQuery *DataSelectQuery) ([]DataCell[T], int) {
	SelectableData := DataSelector[T]{
		GenericDataList: dataList,
		DataSelectQuery: dsQuery,
	}
	// Pipeline is Filter -> Sort -> CollectMetrics -> Paginate
	filtered := SelectableData.Filter()
	filteredTotal := len(filtered.GenericDataList)
	processed := filtered.Sort().Paginate()
	return processed.GenericDataList, filteredTotal
}

// GenericDataSelect takes a list of GenericDataCells and DataSelectQuery and returns selected data as instructed by dsQuery.
func GenericDataSelectWithMetrics[T any](dataList []DataCell[T], dsQuery *DataSelectQuery,
	cachedResources *metricapi.CachedResources, metricClient metricapi.MetricClient) (
	[]DataCell[T], metricapi.MetricPromises) {
	SelectableData := DataSelector[T]{
		GenericDataList: dataList,
		DataSelectQuery: dsQuery,
		CachedResources: cachedResources,
	}
	// Pipeline is Filter -> Sort -> CollectMetrics -> Paginate
	processed := SelectableData.Sort().GetCumulativeMetrics(metricClient).Paginate()
	return processed.GenericDataList, processed.CumulativeMetricsPromises
}

// GenericDataSelect takes a list of GenericDataCells and DataSelectQuery and returns selected data as instructed by dsQuery.
func GenericDataSelectWithFilterAndMetrics[T any](dataList []DataCell[T], dsQuery *DataSelectQuery,
	cachedResources *metricapi.CachedResources, metricClient metricapi.MetricClient) (
	[]DataCell[T], metricapi.MetricPromises, int) {
	SelectableData := DataSelector[T]{
		GenericDataList: dataList,
		DataSelectQuery: dsQuery,
		CachedResources: cachedResources,
	}
	// Pipeline is Filter -> Sort -> CollectMetrics -> Paginate
	filtered := SelectableData.Filter()
	filteredTotal := len(filtered.GenericDataList)
	processed := filtered.Sort().GetCumulativeMetrics(metricClient).Paginate()
	return processed.GenericDataList, processed.CumulativeMetricsPromises, filteredTotal
}

// PodListMetrics returns metrics for every resource on the dataList without aggregating data.
func PodListMetrics[T any](dataList []DataCell[T], dsQuery *DataSelectQuery,
	metricClient metricapi.MetricClient) metricapi.MetricPromises {
	selectableData := DataSelector[T]{
		GenericDataList: dataList,
		DataSelectQuery: dsQuery,
		CachedResources: metricapi.NoResourceCache,
	}

	processed := selectableData.GetMetrics(metricClient)
	return processed.MetricsPromises
}
