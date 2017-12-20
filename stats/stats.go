package stats

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                     Copyright (c) 2009-2017 ESSENTIAL KAOS                         //
//        Essential Kaos Open Source License <https://essentialkaos.com/ekol>         //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"math"
	"sort"
)

// ////////////////////////////////////////////////////////////////////////////////// //

// Data contains stats data
type Data []uint64

func (s Data) Len() int           { return len(s) }
func (s Data) Less(i, j int) bool { return s[i] < s[j] }
func (s Data) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

// ////////////////////////////////////////////////////////////////////////////////// //

// Sort sort measurements data
func (d Data) Sort() {
	if len(d) == 0 {
		return
	}

	sort.Sort(d)
}

// Sum calculate sum of all measurements
func (d Data) Sum() uint64 {
	if len(d) == 0 {
		return 0
	}

	var sum uint64

	for _, v := range d {
		sum += v
	}

	return sum
}

// ////////////////////////////////////////////////////////////////////////////////// //

// Min return minimum value in slice
func Min(d Data) uint64 {
	if len(d) == 0 {
		return 0
	}

	return d[0]
}

// Max return maximum value in slice
func Max(d Data) uint64 {
	return d[len(d)-1]
}

// Mean return average value
func Mean(d Data) uint64 {
	return d.Sum() / uint64(len(d))
}

// StandardDeviation return amount of variation in the dataset
func StandardDeviation(d Data) uint64 {
	m := Mean(d)

	var variance int64

	for _, v := range d {
		variance += (int64(v) - int64(m)) * (int64(v) - int64(m))
	}

	vr := float64(variance / int64(len(d)))

	return uint64(math.Pow(vr, 0.5))
}

// Percentile calcualte percetile
func Percentile(d Data, percent float64) uint64 {
	if percent > 100 {
		return 0
	}

	index := (percent / 100.0) * float64(len(d))

	if index == float64(int64(index)) {
		return d[int(index-1)]
	}

	return d[int(index)]
}
